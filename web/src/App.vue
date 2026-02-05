<script setup>
import { ref, onMounted, nextTick } from 'vue'

const messages = ref([])
const inputText = ref('')
const messagesContainer = ref(null)
const selectedImage = ref(null)
const fileInput = ref(null)
const pendingFiles = ref([])
const showEmojiPicker = ref(false)

const emojis = ['😀', '😃', '😄', '😁', '😅', '😂', '🤣', '😊', '😇', '🙂', '🙃', '😉', '😌', '😍', '🥰', '😘', '😗', '😙', '😚', '😋', '😛', '😝', '😜', '🤪', '🤨', '🧐', '🤓', '😎', '🤩', '🥳', '😏', '😒', '😞', '😔', 'worried', '😕', '🙁', '☹️', '😣', '😖', 'tj', '😩', '🥺', 'cry', '😭', '😤', 'angry', '😡', '🤬', '🤯', 'flush', 'hot', 'cold', 'scream', 'fear', 'ice', 'sweat', 'hug', 'think', 'peep', 'shus', 'lie', 'no_mouth', 'neutral', 'expressionless', 'grim', 'roll', 'surprised', 'frown', 'open_mouth', 'anguished', 'fearful', 'weary', 'sleepy', 'tired', 'yawn', 'mask', 'sick', 'vomit', 'sneeze', 'hot_face', 'cold_face', 'woozy', 'dizzy', 'explode', 'cowboy', 'partying', 'sunglasses', 'nerd', 'monocle', 'confused', 'worried', 'slightly_frowning', 'frowning', 'open_mouth', 'hushed', 'astonished', 'flushed', 'pleading', 'frowning_open', 'anguished', 'fearful', 'cold_sweat', 'disappointed_relieved', 'drooling', 'sob', 'scream', 'confounded', 'persevere', 'disappointed', 'sweat_smile', 'call_me', 'ok_hand', 'thumbs_up', 'thumbs_down', 'clap', 'pray', 'handshake', 'heart', 'broken_heart', 'sparkles', 'fire', 'poop', 'alien', 'clown', 'ghost']
const commonEmojis = ['😀', '😂', '😍', '😭', '😠', '👍', '👎', '🎉', '❤️', '🔥', '👋', '🤔', '👀', '✨']

const openLightbox = (url) => {
  selectedImage.value = url
}

const closeLightbox = () => {
  selectedImage.value = null
}

const toggleEmojiPicker = () => {
  showEmojiPicker.value = !showEmojiPicker.value
}

const addEmoji = (emoji) => {
  inputText.value += emoji
  showEmojiPicker.value = false
}

const triggerFileSelect = () => {
  fileInput.value.click()
}

const handleFileUpload = async (event) => {
  const files = Array.from(event.target.files)
  if (files.length > 0) {
    // In a real app, we would upload these or read them as data URLs for preview
    // For now we just store the file objects to show pending count/names
    const formData = new FormData()

    for (const file of files) {
      formData.append(file.name, file)
    }
    let id = 'todo';
    let basePath = '';
    let access = localStorage.getItem('access-token');
    if (location.pathname === '/') {
      basePath = 'https://dev.webitel.com'
    } else {
      basePath = location.origin;
    }
    let url = `${basePath}/api/storage/file/${id}/upload?channel=chat`


    const res = await fetch(url, {
      method: 'POST',
      body: formData,
      headers: {
        'X-Webitel-Access': access,
      }
      // IMPORTANT: Do NOT manually set Content-Type for FormData with files.
      // The browser sets the correct 'multipart/form-data' boundary automatically.
    })

    const storeFiles = await res.json()

    storeFiles.forEach(f => {
       pendingFiles.value.push({
         name: f.name,
         size: f.size,
         type: f.mime,
         id: f.id,
         link: basePath + f.shared
       })
    })

  }
  // Reset input so same file can be selected again if needed
  if (fileInput.value) fileInput.value.value = ''
}

const removePendingFile = (index) => {
  pendingFiles.value.splice(index, 1)
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
  if (!inputText.value.trim() && pendingFiles.value.length === 0) return

  const text = inputText.value
  inputText.value = ''
  
  // Clear pending files for now as we don't have backend support implemented in this view
  const filesToSend = [...pendingFiles.value]
  pendingFiles.value = []
  showEmojiPicker.value = false
  
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
      body: JSON.stringify({ text, filesToSend })
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

      <div class="input-area-wrapper">
        <div v-if="pendingFiles.length" class="pending-files">
          <div v-for="(file, index) in pendingFiles" :key="index" class="pending-file">
            <span class="file-name">{{ file.name }}</span>
            <span class="remove-file" @click="removePendingFile(index)">×</span>
          </div>
        </div>
        
        <div class="input-area">
          <input 
            type="file" 
            ref="fileInput" 
            multiple 
            style="display: none" 
            @change="handleFileUpload" 
          />
          
          <button class="icon-btn" @click="triggerFileSelect" title="Attach file">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
            </svg>
          </button>

          <input 
            v-model="inputText" 
            @keyup.enter="sendMessage" 
            placeholder="Type a message..."
            type="text"
          />
          
          <div class="emoji-wrapper">
            <button class="icon-btn" @click="toggleEmojiPicker" title="Add emoji">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M14.828 14.828a4 4 0 01-5.656 0M9 10h.01M15 10h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
            </button>
            <div v-if="showEmojiPicker" class="emoji-picker">
              <span v-for="emoji in commonEmojis" :key="emoji" @click="addEmoji(emoji)" class="emoji-item">
                {{ emoji }}
              </span>
            </div>
          </div>

          <button class="send-btn" @click="sendMessage">
            <svg viewBox="0 0 24 24" fill="currentColor" class="send-icon">
              <path d="M2.01 21L23 12 2.01 3 2 10l15 2-15 2z"></path>
            </svg>
          </button>
        </div>
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

.input-area-wrapper {
  display: flex;
  flex-direction: column;
  background: rgba(15, 23, 42, 0.3);
  border-top: 1px solid rgba(255, 255, 255, 0.05);
}

.pending-files {
  padding: 10px 20px 0;
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.pending-file {
  background: rgba(255, 255, 255, 0.1);
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 0.8rem;
  display: flex;
  align-items: center;
  gap: 6px;
}

.remove-file {
  cursor: pointer;
  opacity: 0.7;
  font-weight: bold;
}

.remove-file:hover {
  opacity: 1;
}

.input-area {
  padding: 20px;
  display: flex;
  gap: 10px;
  align-items: center;
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

.icon-btn {
  background: transparent;
  border: none;
  width: 40px;
  height: 40px;
  border-radius: 10px;
  color: var(--text-dim);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
  padding: 8px;
}

.icon-btn:hover {
  background: rgba(255, 255, 255, 0.1);
  color: var(--text-main);
  transform: none;
  box-shadow: none;
}

.send-btn {
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

.send-btn:hover {
  background: #2563eb;
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(37, 99, 235, 0.4);
}

.send-btn:active {
  transform: translateY(0);
}

.send-icon {
  width: 20px;
  height: 20px;
}

.emoji-wrapper {
  position: relative;
}

.emoji-picker {
  position: absolute;
  bottom: 50px;
  right: 0;
  background: #1e293b;
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 12px;
  padding: 10px;
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  gap: 5px;
  box-shadow: 0 4px 20px rgba(0,0,0,0.5);
  z-index: 10;
  width: 280px;
}

.emoji-item {
  cursor: pointer;
  padding: 5px;
  border-radius: 4px;
  text-align: center;
  font-size: 1.2rem;
  transition: background 0.2s;
}

.emoji-item:hover {
  background: rgba(255, 255, 255, 0.1);
}
</style>
