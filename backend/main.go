// エントリーポイント
/*
package main

import "fmt"

func main() {
	fmt.Println("Helloworld")
}
*/

/*
package main

import (
	"github.com/gin-gonic/gin"
	//"net/http"
)

func main() {
	//Ginエンジンのインスタンス作成
	r := gin.Default()

	//ルートURL（”/"）に対するGETリクエストのルート
	r.GET("/", func(c *gin.Context) {
		//JSONレスポンスを返す
		c.JSON(200, gin.H{
			"message": "hello world",
		})
	})
	r.Run(":8080")
}
*/

//http://localhost:8080/

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	speech "cloud.google.com/go/speech/apiv1"
	"cloud.google.com/go/speech/apiv1/speechpb"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"google.golang.org/api/option"
)

// 動画の情報を表す構造体
type Video struct {
	ID         string `json:"id"`
	YoutubeUrl string `json:"youtube_url"`
	AudioPath  string `json:"audio_url"`
	Status     string `json:"status"`
	CreatedAt  string `json:"created_at"`
	UpdateAt   string `json:"update_at"`
}

// Google Speech-to-Text使用量追跡構造体
type SpeechUsage struct {
	Month       string `json:"month"`        // YYYY-MM形式
	UsedMinutes int    `json:"used_minutes"` // 使用分数
}

// 字幕（文字起こし）の情報を表す構造体
type Transcript struct {
	ID           string `json:"id"`
	VideoId      string `json:"video_id"`
	Language     string `json:"langiage"`
	TransriptSrt string `json:"transript_srt"`
	CreatedAt    string `json:"created_at"`
}

// 翻訳済み字幕情報を表す構造体
type Translation struct {
	ID            string `json:"id"`
	TranscriptId  string `json:"transcript_id"`
	SourceLang    string `json:"source_lang"`
	TargetLang    string `json:"target_lang"`
	TranslatedSrt string `json:"translated_srt"`
	ModelUsed     string `json:"model_used"`
	CreatedAt     string `json:"created_at"`
}

// 翻訳結果構造体
type TranslationResult struct {
	TranslatedText string `json:"translated_text"`
	Language       string `json:"language"`
	Timestamp      string `json:"timestamp"`
	Status         string `json:"status"`
}

// スライス（配列）（DBのテーブル代わりメモリ上に置くためサーバー停止後消える）
var (
	videos       = []Video{}       //動画情報テーブル
	transcripts  = []Transcript{}  //字幕情報テーブル
	translations = []Translation{} //翻訳情報テーブル
	mu           sync.Mutex        // 複数の処理が同時にデータを書き換えるのを防ぐためのロック
)

// 翻訳文字数管理
var (
	monthlyCharCount int
	monthlyStart     = time.Now()
	limit            = 400000 // 月40万文字
	muChar           sync.Mutex
)

// Google Speech-to-Text使用時間管理（月間60分制限）
var (
	speechUsageMinutes int
	speechUsageStart   = time.Now()
	speechLimit        = 60 // 月60分
	muSpeech           sync.Mutex
)

func canTranslate(text string) bool {
	muChar.Lock()
	defer muChar.Unlock()

	// 月が変わったらリセット（30日基準）
	if time.Since(monthlyStart).Hours() > 24*30 {
		monthlyCharCount = 0
		monthlyStart = time.Now()
	}

	chars := len([]rune(text))
	if monthlyCharCount+chars > limit {
		return false
	}
	monthlyCharCount += chars
	return true
}

// Google Speech-to-Text使用可能かチェック（音声時間分数）
func canUseSpeechToText(audioDurationMinutes int) bool {
	muSpeech.Lock()
	defer muSpeech.Unlock()

	// 月が変わったらリセット（30日基準）
	if time.Since(speechUsageStart).Hours() > 24*30 {
		speechUsageMinutes = 0
		speechUsageStart = time.Now()
	}

	return speechUsageMinutes+audioDurationMinutes <= speechLimit
}

// Google Speech-to-Text使用量を更新
func updateSpeechUsage(audioDurationMinutes int) {
	muSpeech.Lock()
	defer muSpeech.Unlock()

	speechUsageMinutes += audioDurationMinutes
}
func main() {
	// 環境変数を読み込み
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found")
	}

	// Ginルーターを設定
	router := gin.Default()

	// CORSの設定
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"}, // Reactのアドレス
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: true,
	}))

	// ルートを設定
	router.GET("/videos", getVideos)
	router.POST("/videos", createVideo)
	router.GET("/videos/:id", getVideo)
	router.PUT("/videos/:id/status", updateVideoStatusHandler)
	router.GET("/videos/:id/transcript", getTranscript)
	router.GET("/videos/:id/translation", getTranslation)

	log.Println("Server started at :8080")
	router.Run(":8080")
}

// GET /videos - 全動画取得
func getVideos(c *gin.Context) {
	mu.Lock()
	defer mu.Unlock()
	c.JSON(http.StatusOK, videos)
}

// POST /videos - 新規動画作成
func createVideo(c *gin.Context) {
	var req struct {
		YoutubeURL string `json:"youtube_url" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 新しい動画を作成
	video := Video{
		ID:         uuid.New().String(),
		YoutubeUrl: req.YoutubeURL,
		Status:     "processing",
		CreatedAt:  time.Now().Format(time.RFC3339),
		UpdateAt:   time.Now().Format(time.RFC3339),
	}

	mu.Lock()
	videos = append(videos, video)
	mu.Unlock()

	// バックグラウンド処理を開始
	go processVideo(video, os.Getenv("GEMINI_API_KEY"))

	c.JSON(http.StatusCreated, video)
}

// GET /videos/:id - 特定動画取得
func getVideo(c *gin.Context) {
	id := c.Param("id")

	mu.Lock()
	defer mu.Unlock()

	for _, video := range videos {
		if video.ID == id {
			c.JSON(http.StatusOK, video)
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Video not found"})
}

// PUT /videos/:id/status - 動画ステータス更新
func updateVideoStatusHandler(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Status string `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updateVideoStatus(id, req.Status)
	c.JSON(http.StatusOK, gin.H{"message": "Status updated"})
}

// GET /videos/:id/transcript - 字幕取得
func getTranscript(c *gin.Context) {
	id := c.Param("id")

	mu.Lock()
	defer mu.Unlock()

	for _, transcript := range transcripts {
		if transcript.VideoId == id {
			c.JSON(http.StatusOK, transcript)
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Transcript not found"})
}

// GET /videos/:id/translation - 翻訳取得
func getTranslation(c *gin.Context) {
	id := c.Param("id")

	mu.Lock()
	defer mu.Unlock()

	for _, translation := range translations {
		if translation.TranscriptId == id {
			c.JSON(http.StatusOK, translation)
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Translation not found"})
}

// 動画ステータス更新ヘルパー関数
func updateVideoStatus(videoID, status string) {
	mu.Lock()
	defer mu.Unlock()

	for i, video := range videos {
		if video.ID == videoID {
			videos[i].Status = status
			videos[i].UpdateAt = time.Now().Format(time.RFC3339)
			break
		}
	}
}

// バックグラウンド処理
func processVideo(v Video, apiKey string) {
	log.Printf("処理開始: VideoID=%s", v.ID)
	audioFile := v.ID + ".mp3"

	// 1. yt-dlpで音声抽出
	log.Printf("yt-dlp開始: %s", v.YoutubeUrl)
	cmdYtdlp := exec.Command(
		"yt-dlp",
		"-x",
		"--audio-format", "mp3",
		"-o", audioFile,
		v.YoutubeUrl,
	)
	if err := cmdYtdlp.Run(); err != nil {
		updateVideoStatus(v.ID, "error")
		log.Println("yt-dlp error:", err)
		return
	}
	log.Printf("yt-dlp完了: %s", audioFile)

	// 2. Google Speech-to-Textで文字起こし
	log.Printf("Google Speech-to-Text開始: %s", audioFile)

	// 音声時間を推定（簡易実装：ファイルサイズから推定）
	audioInfo, err := os.Stat(audioFile)
	if err != nil {
		updateVideoStatus(v.ID, "error")
		log.Printf("音声ファイル情報取得エラー: %v", err)
		return
	}

	// 簡易推定：1MB ≈ 1分の音声（実際はもっと複雑）
	estimatedMinutes := int(audioInfo.Size() / (1024 * 1024))
	if estimatedMinutes < 1 {
		estimatedMinutes = 1 // 最低1分として計算
	}

	// 使用制限チェック
	if !canUseSpeechToText(estimatedMinutes) {
		updateVideoStatus(v.ID, "error")
		log.Printf("Google Speech-to-Text月間制限（60分）を超過: 推定%d分", estimatedMinutes)
		return
	}

	transcriptText, err := transcribeWithGoogleSpeech(audioFile)
	if err != nil {
		updateVideoStatus(v.ID, "error")
		log.Printf("Google Speech-to-Text error: %v", err)
		return
	}

	// 使用量を更新
	updateSpeechUsage(estimatedMinutes)
	log.Printf("Google Speech-to-Text完了: 文字数=%d, 使用時間=%d分", len(transcriptText), estimatedMinutes)

	// 3. GPT翻訳
	log.Printf("翻訳開始: %d文字", len(transcriptText))
	_, err = translateTextWithGPT(transcriptText, apiKey)
	if err != nil {
		updateVideoStatus(v.ID, "error")
		log.Println("translation error:", err)
		return
	}
	log.Printf("翻訳完了")

	// 4. 結果保存
	t := Transcript{
		ID:           uuid.New().String(),
		VideoId:      v.ID,
		Language:     "en",
		TransriptSrt: transcriptText,
		CreatedAt:    time.Now().Format(time.RFC3339),
	}
	mu.Lock()
	transcripts = append(transcripts, t)
	updateVideoStatus(v.ID, "completed")
	mu.Unlock()
	log.Printf("保存完了: VideoID=%s", v.ID)
}

// 翻訳関数（Gemini API利用）
func translateTextWithGPT(text, apiKey string) (*TranslationResult, error) {
	if !canTranslate(text) {
		return nil, fmt.Errorf("翻訳上限超えました（40万文字/月）")
	}

	payload := map[string]interface{}{
		"model": "gemini-1.5-flash",
		"messages": []map[string]string{
			{"role": "system", "content": "You are a professional translator."},
			{"role": "user", "content": "Translate the following text to English:\n" + text},
		},
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash-latest:generateContent?key="+apiKey, bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var res map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	choices := res["choices"].([]interface{})
	content := choices[0].(map[string]interface{})["message"].(map[string]interface{})["content"].(string)

	result := &TranslationResult{
		TranslatedText: content,
		Language:       "en",
		Timestamp:      time.Now().Format(time.RFC3339),
		Status:         "completed",
	}

	return result, nil
}

// Google Speech-to-Textで音声ファイルを文字起こしする関数
func transcribeWithGoogleSpeech(audioFile string) (string, error) {
	ctx := context.Background()

	// 認証情報をJSON文字列から設定
	credentialsJSON := os.Getenv("GOOGLE_CREDENTIALS_JSON")
	if credentialsJSON == "" {
		return "", fmt.Errorf("GOOGLE_CREDENTIALS_JSON環境変数が設定されていません")
	}

	// エスケープされた改行文字を実際の改行文字に変換
	credentialsJSON = strings.ReplaceAll(credentialsJSON, "\\n", "\n")
	
	// JSONの妥当性を確認
	var credMap map[string]interface{}
	if err := json.Unmarshal([]byte(credentialsJSON), &credMap); err != nil {
		return "", fmt.Errorf("認証JSON解析エラー: %v", err)
	}

	credentialsBytes := []byte(credentialsJSON)

	// クライアントを作成
	client, err := speech.NewClient(ctx, option.WithCredentialsJSON(credentialsBytes))
	if err != nil {
		return "", fmt.Errorf("Speech-to-Textクライアント作成エラー: %v", err)
	}
	defer client.Close()

	// 音声ファイルを読み込み
	audioData, err := os.Open(audioFile)
	if err != nil {
		return "", fmt.Errorf("音声ファイル読み込みエラー: %v", err)
	}
	defer audioData.Close()

	data, err := io.ReadAll(audioData)
	if err != nil {
		return "", fmt.Errorf("音声データ読み込みエラー: %v", err)
	}

	// 音声認識リクエストを作成
	req := &speechpb.RecognizeRequest{
		Config: &speechpb.RecognitionConfig{
			Encoding:        speechpb.RecognitionConfig_MP3, // MP3形式
			SampleRateHertz: 44100,                          // サンプルレート
			LanguageCode:    "en-US",                        // 言語設定
		},
		Audio: &speechpb.RecognitionAudio{
			AudioSource: &speechpb.RecognitionAudio_Content{
				Content: data,
			},
		},
	}

	// 音声認識を実行
	resp, err := client.Recognize(ctx, req)
	if err != nil {
		return "", fmt.Errorf("音声認識エラー: %v", err)
	}

	// 結果をテキストに変換
	var transcriptText string
	for _, result := range resp.Results {
		for _, alt := range result.Alternatives {
			transcriptText += alt.Transcript + " "
		}
	}

	return transcriptText, nil
}
