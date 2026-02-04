<script setup>
import { ref, onMounted, nextTick } from 'vue'

const messages = ref([])
const inputText = ref('')
const messagesContainer = ref(null)
const selectedImage = ref(null)

const openLightbox = (url) => {
  selectedImage.value = url
}

const closeLightbox = () => {
  selectedImage.value = null
}

const connectWebSocket = () => {
  let wsUrl = `${window.location.href.replace(/http/, 'ws')}`;
  if (window.location.pathname !== '/') {
    wsUrl+= '/'
  }
  wsUrl+='ws';
  const ws = new WebSocket(wsUrl)
  console.error('try open: ', wsUrl)
  
  ws.onmessage = (event) => {
    try {
      const data = JSON.parse(event.data)
      // Expecting wrapped message or raw structure. 
      // Based on main.go logic: srv.Broadcast(msg.Body)
      /* 
         Structure from main.go:
         type MessageWrapper struct {
            ID       string  `json:"ID"`
            Message  Message `json:"message"`
            UserID   string  `json:"user_id"`
            DomainID int64   `json:"domain_id"`
         }
      */
      
      let msg = {
        id: Date.now(),
        text: 'Unknown message',
        isMine: false,
        from: 'User'
      }

      // Check if message is from "2522" (hardcoded user ID)
      // Structure: data.message.from.sub (or id)
      let fromID = ""
      let fromName = "User"
      let text = ""
      let id = Date.now()
      let me = false

      if (data.message) {
          text = data.message.text
          if (data.message.from) {
             fromID = data.message.from.sub || data.message.from.id
             fromName = data.message.from.name || fromID
          }
          me = data.message.me
          if (data.ID) id = data.ID
      } else if (data.text) {
          text = data.text
      } else {
          // Fallback
          text = JSON.stringify(data)
      }
      
      msg.id = id
      msg.text = text
      msg.id = id
      msg.text = text
      msg.from = fromName
      
      // Handle images/files
      if (data.message && data.message.images && Array.isArray(data.message.images)) {
        msg.files = data.message.images.map(img => ({
          id: img.id,
          url: img.url,
          name: img.file_name,
          type: img.mime_type
        }))
      } else {
        msg.files = []
      }
      
      // Use backend 'me' flag if available
      if (me) {
          msg.isMine = true
          msg.from = "Me"
      }
      console.error(data)
      if (data.message && data.message.created_at) {
          msg.createdAt = data.message.created_at
      } else {
          msg.createdAt = Date.now()
      }

      messages.value.push(msg)
      scrollToBottom()
    } catch (e) {
      console.error("Parse error", e)
    }
  }

  ws.onclose = () => {
    console.log("WS closed, retrying...")
    setTimeout(connectWebSocket, 3000)
  }
}

const formatTime = (timestamp) => {
  if (!timestamp) return ''
  const date = new Date(timestamp)
  return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

const sendMessage = async () => {
  if (!inputText.value.trim()) return

  const text = inputText.value
  inputText.value = ''
  
  // Optimistic UI removed to prevent duplication as per user feedback
  // We rely on the WebSocket broadcast to show the message.
  let apiUrl = window.location.href
  if (!/\/$/.test(apiUrl)) {
    apiUrl += '/'
  }
  
  apiUrl += 'api/messages'

  try {
    const res = await fetch(apiUrl, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'access-token': localStorage.getItem('access-token') || ''
      },
      body: JSON.stringify({ text })
    })
    if (!res.ok) {
        throw new Error("Failed to send")
    }
  } catch (e) {
    console.error(e)
    messages.value.push({
        id: Date.now(),
        text: "Error sending message: " + e.message,
        isMine: true,
        from: "System"
    })
  }
}

const scrollToBottom = async () => {
  await nextTick()
  if (messagesContainer.value) {
    messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
  }
}

onMounted(() => {
  connectWebSocket()
})
</script>

<template>
  <div class="app-container">
    <div class="chat-window">
      <header>
        <h1>Webitel Chat - Preview</h1>
        <div class="status-indicator"></div>
      </header>
      
      <div class="messages" ref="messagesContainer">
        <div 
          v-for="msg in messages" 
          :key="msg.id" 
          class="message-bubble"
          :class="{ 'mine': msg.isMine, 'system': msg.from === 'System' }"
        >
          <div class="sender" v-if="!msg.isMine">{{ msg.from }}</div>
          <div class="text">{{ msg.text }}</div>
          
          <div v-if="msg.files && msg.files.length" class="attachments">
            <div v-for="file in msg.files" :key="file.id" class="attachment-item">
               <img v-if="file.type.startsWith('image/')" :src="file.url" :alt="file.name" class="message-media clickable" @click="openLightbox(file.url)" />
               <video v-else-if="file.type.startsWith('video/')" :src="file.url" controls class="message-media"></video>
               <a v-else :href="file.url" target="_blank" class="file-link">{{ file.name || 'Download File' }}</a>
            </div>
          </div>
          <div class="time">{{ formatTime(msg.createdAt) }}</div>
        </div>
      </div>

      <div class="input-area">
        <input 
          v-model="inputText" 
          @keyup.enter="sendMessage" 
          placeholder="Type a message..."
          type="text"
        />
        <button @click="sendMessage">
          <svg viewBox="0 0 24 24" fill="currentColor" class="send-icon">
            <path d="M2.01 21L23 12 2.01 3 2 10l15 2-15 2z"></path>
          </svg>
        </button>
      </div>
    </div>
    
    <!-- Lightbox Modal -->
    <div v-if="selectedImage" class="lightbox" @click="closeLightbox">
      <div class="lightbox-content">
        <img :src="selectedImage" />
      </div>
    </div>
  </div>
</template>

<style>
/* CSS Reset & Variables */
:root {
  --bg-color: #0f172a;
  --panel-bg: rgba(30, 41, 59, 0.7);
  --primary: #3b82f6;
  --primary-glow: rgba(59, 130, 246, 0.5);
  --text-main: #f1f5f9;
  --text-dim: #94a3b8;
  --bubble-incoming: #334155;
  --bubble-outgoing: #2563eb;
  --bubble-system: #ef4444;
}

body {
  margin: 0;
  background-color: var(--bg-color);
  color: var(--text-main);
  font-family: 'Inter', sans-serif;
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  background: linear-gradient(135deg, #0f172a 0%, #1e293b 100%);
}

.app-container {
  width: 90vw;
  max-width: 1000px;
  height: 90vh;
  display: flex;
  flex-direction: column;
}

.chat-window {
  background: var(--panel-bg);
  backdrop-filter: blur(12px);
  border-radius: 20px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5);
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

header {
  padding: 20px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
  display: flex;
  align-items: center;
  justify-content: space-between;
}

header h1 {
  margin: 0;
  font-size: 1.2rem;
  font-weight: 600;
  background: linear-gradient(to right, #60a5fa, #a78bfa);
  background-clip: text;
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
}

.status-indicator {
  width: 10px;
  height: 10px;
  background-color: #10b981;
  border-radius: 50%;
  box-shadow: 0 0 10px #10b981;
}

.messages {
  flex: 1;
  padding: 20px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 15px;
}

.message-bubble {
  max-width: 80%;
  padding: 12px 16px;
  border-radius: 16px;
  font-size: 0.95rem;
  line-height: 1.4;
  position: relative;
  animation: popIn 0.3s cubic-bezier(0.175, 0.885, 0.32, 1.275);
}

@keyframes popIn {
  from { opacity: 0; transform: scale(0.9) translateY(10px); }
  to { opacity: 1; transform: scale(1) translateY(0); }
}

.message-bubble.mine {
  align-self: flex-end;
  background: var(--bubble-outgoing);
  color: white;
  border-bottom-right-radius: 4px;
  box-shadow: 0 4px 15px rgba(37, 99, 235, 0.3);
}

.message-bubble:not(.mine) {
  align-self: flex-start;
  background: var(--bubble-incoming);
  color: var(--text-main);
  border-bottom-left-radius: 4px;
}
.message-bubble.system {
    background: var(--bubble-system);
    align-self: center;
}

.sender {
  font-size: 0.7rem;
  color: var(--text-dim);
  margin-bottom: 4px;
}

.attachments {
  margin-top: 8px;
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.message-media {
  max-width: 100%;
  max-height: 300px;
  border-radius: 8px;
  object-fit: cover;
}

.message-media.clickable {
  cursor: zoom-in;
  transition: transform 0.2s;
}

.message-media.clickable:hover {
  transform: scale(1.02);
}

/* Lightbox */
.lightbox {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background: rgba(0, 0, 0, 0.9);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1000;
  animation: fadeIn 0.2s ease-out;
}

.lightbox-content img {
  max-width: 90vw;
  max-height: 90vh;
  border-radius: 8px;
  box-shadow: 0 0 20px rgba(0,0,0,0.5);
  animation: zoomIn 0.3s cubic-bezier(0.175, 0.885, 0.32, 1.275);
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

@keyframes zoomIn {
  from { transform: scale(0.9); opacity: 0; }
  to { transform: scale(1); opacity: 1; }
}

.file-link {
  display: inline-block;
  padding: 6px 10px;
  background: rgba(255, 255, 255, 0.1);
  border-radius: 6px;
  color: inherit;
  text-decoration: none;
  font-size: 0.85rem;
}

.file-link:hover {
  background: rgba(255, 255, 255, 0.2);
}

.time {
  font-size: 0.65rem;
  color: rgba(255, 255, 255, 0.7);
  text-align: right;
  margin-top: 4px;
}

.input-area {
  padding: 20px;
  background: rgba(15, 23, 42, 0.3);
  display: flex;
  gap: 10px;
  border-top: 1px solid rgba(255, 255, 255, 0.05);
}

input {
  flex: 1;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.1);
  padding: 12px 16px;
  border-radius: 12px;
  color: white;
  outline: none;
  transition: all 0.2s;
}

input:focus {
  background: rgba(255, 255, 255, 0.1);
  border-color: var(--primary);
  box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.2);
}

button {
  background: var(--primary);
  border: none;
  width: 48px;
  height: 48px;
  border-radius: 12px;
  color: white;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}

button:hover {
  background: #2563eb;
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(37, 99, 235, 0.4);
}

button:active {
  transform: translateY(0);
}

.send-icon {
  width: 20px;
  height: 20px;
}
</style>
