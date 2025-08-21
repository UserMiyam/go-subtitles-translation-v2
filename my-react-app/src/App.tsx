import { useState, useEffect } from 'react'

// 字幕データの型定義
interface SubtitleSegment {
  start_time: number
  end_time: number
  text: string
}

interface Transcript {
  id: string
  video_id: string
  language: string
  transcript_srt: string
  segments: SubtitleSegment[]
  created_at: string
}

interface Video {
  id: string
  youtube_url: string
  status: string
  created_at: string
  update_at: string
}

function App() {
  // YouTubeのURLを保存する場所
  const [url, setUrl] = useState('')
  
  // 今処理中かどうかを保存する場所
  const [loading, setLoading] = useState(false)
  
  // メッセージを保存する場所
  const [message, setMessage] = useState('')
  
  // 動画リストを保存する場所
  const [videos, setVideos] = useState<Video[]>([])

  // 動画リストを取得する関数
  function loadVideos() {
    const controller = new AbortController()
    const timeoutId = setTimeout(() => controller.abort(), 5000) // 5秒タイムアウト
    
    fetch('http://localhost:8080/videos', { 
      signal: controller.signal
    })
      .then(response => {
        clearTimeout(timeoutId)
        if (!response.ok) {
          throw new Error(`HTTP ${response.status}`)
        }
        return response.json()
      })
      .then(data => {
        setVideos(Array.isArray(data) ? data : [])
      })
      .catch(error => {
        clearTimeout(timeoutId)
        console.error('動画リスト取得エラー:', error)
        setVideos([]) // エラー時は空配列にする
        setMessage('サーバーに接続できません。Goサーバーが起動しているか確認してください。')
      })
  }

  // SRTファイルを生成する関数
  function generateSRT(segments: SubtitleSegment[]): string {
    let srtContent = ''
    
    segments.forEach((segment, index) => {
      const startTime = formatSRTTime(segment.start_time)
      const endTime = formatSRTTime(segment.end_time)
      
      srtContent += `${index + 1}\n`
      srtContent += `${startTime} --> ${endTime}\n`
      srtContent += `${segment.text}\n\n`
    })
    
    return srtContent
  }

  // SRT時間フォーマット関数
  function formatSRTTime(seconds: number): string {
    const hours = Math.floor(seconds / 3600)
    const minutes = Math.floor((seconds % 3600) / 60)
    const secs = Math.floor(seconds % 60)
    const millisecs = Math.floor((seconds % 1) * 1000)
    
    return `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')},${millisecs.toString().padStart(3, '0')}`
  }

  // SRTファイルをダウンロードする関数
  async function downloadSRT(videoId: string) {
    try {
      const response = await fetch(`http://localhost:8080/videos/${videoId}/transcript`)
      const transcript: Transcript = await response.json()
      
      if (transcript.segments && transcript.segments.length > 0) {
        const srtContent = generateSRT(transcript.segments)
        const blob = new Blob([srtContent], { type: 'text/plain' })
        const url = URL.createObjectURL(blob)
        
        const a = document.createElement('a')
        a.href = url
        a.download = `${videoId}.srt`
        document.body.appendChild(a)
        a.click()
        document.body.removeChild(a)
        URL.revokeObjectURL(url)
      } else {
        alert('字幕データが見つかりません')
      }
    } catch (error) {
      alert('SRTダウンロードエラー: ' + error)
    }
  }

  // コンポーネント初回読み込み時
  useEffect(() => {
    loadVideos()
  }, [])

  // ボタンを押した時の処理
  function handleButtonClick() {
    // URLが空の場合
    if (url === '') {
      setMessage('URLを入力してください')
      return
    }

    // 処理開始
    setLoading(true)
    setMessage('処理中です...')

    // サーバーにデータを送る
    sendToServer()
  }

  // サーバーにデータを送る処理
  function sendToServer() {
    // サーバーのアドレス
    const serverUrl = 'http://localhost:8080/videos'
    
    // 送るデータ
    const data = {
      youtube_url: url
    }

    // videosサーバーに送信React fetchAPI　
    fetch(serverUrl, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(data)
    })
    .then(response => {
      // サーバーからの返事をチェック
      if (response.ok) {
        return response.json()
      } else {
        throw new Error('サーバーエラー')
      }
    })
    .then(data => {
      // 成功した場合
      setMessage('翻訳が開始されましたID: ' + data.id)
      setUrl('') // 入力欄を空にする
      loadVideos() // 動画リストを再読み込み
    })
    .catch(error => {
      // エラーが起きた場合
      setMessage('エラーが発生しました: ' + error.message)
    })
    .finally(() => {
      // 最後に必ず実行される
      setLoading(false)
    })
  }

  return (
    <div style={{ padding: '20px' }}>
      <h1>YouTube字幕翻訳</h1>
      
      {/* URL入力欄 */}
      <div style={{ marginBottom: '10px' }}>
        <label>YouTube URL:</label>
        <br />
        <input
          type="text"
          value={url}
          onChange={(e) => setUrl(e.target.value)}
          placeholder="https://youtube.com/watch?v=..."
          style={{ 
            width: '100%', 
            padding: '10px',
            fontSize: '16px',
            border: '2px solid #ccc',
            borderRadius: '5px',
            outline: 'none'
          }}
          disabled={loading}
        />
      </div>

      {/* ボタン */}
      <button
        onClick={handleButtonClick}
        disabled={loading}
        style={{
          padding: '10px 20px',
          backgroundColor: loading ? 'gray' : 'blue',
          color: 'white',
          border: 'none',
          fontSize: '16px',
          cursor: loading ? 'not-allowed' : 'pointer'
        }}
      >
        {loading ? '処理中...' : '翻訳開始'}
      </button>

      {/* メッセージ表示 */}
      {message && (
        <div style={{ 
          marginTop: '20px',
          padding: '10px',
          backgroundColor: '#f0f0f0',
          border: '1px solid #ccc'
        }}>
          {message}
        </div>
      )}

      {/* 動画リスト表示 */}
      <div style={{ marginTop: '30px' }}>
        <h2>処理済み動画一覧</h2>
        <button 
          onClick={loadVideos}
          style={{
            marginBottom: '10px',
            padding: '5px 10px',
            backgroundColor: 'green',
            color: 'white',
            border: 'none',
            cursor: 'pointer'
          }}
        >
          リスト更新
        </button>
        
        {videos.length === 0 ? (
          <p>処理済みの動画はありません</p>
        ) : (
          <div>
            <p>動画数: {videos.length}</p>
            <div style={{ display: 'grid', gap: '10px' }}>
              {videos.slice(0, 10).map((video) => (
                <div 
                  key={video.id}
                  style={{
                    border: '1px solid #ccc',
                    padding: '15px',
                    borderRadius: '5px',
                    backgroundColor: video.status === 'completed' ? '#f0fff0' : '#fff5ee'
                  }}
                >
                  <div><strong>ID:</strong> {video.id}</div>
                  <div><strong>URL:</strong> <a href={video.youtube_url} target="_blank" rel="noopener noreferrer">{video.youtube_url}</a></div>
                  <div><strong>ステータス:</strong> <span style={{color: video.status === 'completed' ? 'green' : 'orange'}}>{video.status}</span></div>
                  <div><strong>作成日時:</strong> {new Date(video.created_at).toLocaleString()}</div>
                  
                  {video.status === 'completed' && (
                    <div style={{ marginTop: '10px' }}>
                      <button
                        onClick={() => downloadSRT(video.id)}
                        style={{
                          padding: '8px 15px',
                          backgroundColor: '#007bff',
                          color: 'white',
                          border: 'none',
                          borderRadius: '3px',
                          cursor: 'pointer',
                          marginRight: '10px'
                        }}
                      >
                        SRTダウンロード
                      </button>
                    </div>
                  )}
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

export default App