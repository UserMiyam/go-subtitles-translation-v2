import { useState } from 'react'

function App() {
  // YouTubeのURLを保存する場所
  const [url, setUrl] = useState('')
  
  // 今処理中かどうかを保存する場所
  const [loading, setLoading] = useState(false)
  
  // メッセージを保存する場所
  const [message, setMessage] = useState('')

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

    // サーバーに送信
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
      setMessage('翻訳が開始されました！ID: ' + data.id)
      setUrl('') // 入力欄を空にする
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
    </div>
  )
}

export default App