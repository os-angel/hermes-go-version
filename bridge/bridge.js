'use strict'

/**
 * bridge.js — HTTP bridge entre hermes-go y WhatsApp via Baileys.
 *
 * Expone:
 *   GET  /health           -> { status: "ok" }
 *   GET  /status           -> { connected: bool, phone: string }
 *   GET  /messages         -> array de mensajes pendientes (vacia la cola)
 *   POST /send             -> { chatId, message, replyTo? }
 *   POST /send-media       -> { chatId, filePath, mediaType, caption, fileName }
 *   POST /typing           -> { chatId }
 *
 * Argumentos:
 *   --port     Puerto HTTP (default 3001)
 *   --session  Directorio para auth_state de Baileys (default ./session)
 *   --mode     "bot" | "user"  (default "bot")
 */

const { makeWASocket, useMultiFileAuthState, DisconnectReason, fetchLatestBaileysVersion } = require('@whiskeysockets/baileys')
const express = require('express')
const pino = require('pino')
const qrcode = require('qrcode-terminal')
const fs = require('fs')
const path = require('path')

// --- Argumentos ---
const args = parseArgs(process.argv.slice(2))
const PORT = parseInt(args['port'] || '3001', 10)
const SESSION_DIR = args['session'] || path.join(__dirname, 'session')
const MODE = args['mode'] || 'bot'

const logger = pino({ level: 'info' })

// --- Cola de mensajes ---
const messageQueue = []

// --- Estado de conexion ---
let sock = null
let isConnected = false
let phoneNumber = ''

// --- HTTP server ---
const app = express()
app.use(express.json())

app.get('/health', (_req, res) => {
  res.json({ status: 'ok', connected: isConnected })
})

app.get('/status', (_req, res) => {
  res.json({ connected: isConnected, phone: phoneNumber })
})

// GET /messages: retorna y vacia la cola de mensajes pendientes.
app.get('/messages', (_req, res) => {
  const msgs = messageQueue.splice(0, messageQueue.length)
  res.json(msgs)
})

// POST /send: envia un mensaje de texto.
app.post('/send', async (req, res) => {
  const { chatId, message, replyTo } = req.body
  if (!chatId || !message) {
    return res.status(400).json({ error: 'chatId and message required' })
  }
  if (!sock || !isConnected) {
    return res.status(503).json({ error: 'not connected' })
  }
  try {
    const opts = {}
    if (replyTo) {
      opts.quoted = { key: { id: replyTo, remoteJid: chatId } }
    }
    await sock.sendMessage(normalizeJid(chatId), { text: message }, opts)
    res.json({ ok: true })
  } catch (err) {
    logger.error({ err }, 'send error')
    res.status(500).json({ error: err.message })
  }
})

// POST /send-media: envia un archivo (imagen, documento, audio, video).
app.post('/send-media', async (req, res) => {
  const { chatId, filePath, mediaType, caption, fileName } = req.body
  if (!chatId || !filePath) {
    return res.status(400).json({ error: 'chatId and filePath required' })
  }
  if (!sock || !isConnected) {
    return res.status(503).json({ error: 'not connected' })
  }
  try {
    const buffer = fs.readFileSync(filePath)
    const type = resolveMediaType(mediaType, fileName || filePath)
    const msgPayload = {
      [type]: buffer,
      caption: caption || '',
      fileName: fileName || path.basename(filePath),
      mimetype: mediaType || 'application/octet-stream',
    }
    await sock.sendMessage(normalizeJid(chatId), msgPayload)
    res.json({ ok: true })
  } catch (err) {
    logger.error({ err }, 'send-media error')
    res.status(500).json({ error: err.message })
  }
})

// POST /typing: envia el indicador de escritura (composing).
app.post('/typing', async (req, res) => {
  const { chatId } = req.body
  if (!chatId) {
    return res.status(400).json({ error: 'chatId required' })
  }
  if (!sock || !isConnected) {
    return res.json({ ok: true }) // no-op si no conectado
  }
  try {
    await sock.sendPresenceUpdate('composing', normalizeJid(chatId))
    res.json({ ok: true })
  } catch (err) {
    logger.warn({ err }, 'typing error (non-fatal)')
    res.json({ ok: true })
  }
})

// --- Baileys ---
async function startBaileys () {
  if (!fs.existsSync(SESSION_DIR)) {
    fs.mkdirSync(SESSION_DIR, { recursive: true })
  }

  const { state, saveCreds } = await useMultiFileAuthState(SESSION_DIR)
  const { version } = await fetchLatestBaileysVersion()

  logger.info({ version, mode: MODE, port: PORT }, 'baileys starting')

  sock = makeWASocket({
    version,
    auth: state,
    logger: pino({ level: 'silent' }),
    printQRInTerminal: false,
    browser: MODE === 'bot'
      ? ['Hermes', 'Bot', '1.0.0']
      : ['Hermes', 'Chrome', '1.0.0'],
    syncFullHistory: false,
    getMessage: async () => undefined,
  })

  sock.ev.on('creds.update', saveCreds)

  sock.ev.on('connection.update', (update) => {
    const { connection, lastDisconnect, qr } = update

    if (qr) {
      logger.info('scan QR code to authenticate')
      qrcode.generate(qr, { small: true })
    }

    if (connection === 'close') {
      isConnected = false
      const code = lastDisconnect?.error?.output?.statusCode
      const shouldReconnect = code !== DisconnectReason.loggedOut
      logger.warn({ code, reconnect: shouldReconnect }, 'connection closed')
      if (shouldReconnect) {
        setTimeout(startBaileys, 3000)
      }
    }

    if (connection === 'open') {
      isConnected = true
      phoneNumber = sock.user?.id?.split(':')[0] || ''
      logger.info({ phone: phoneNumber }, 'connected')
    }
  })

  sock.ev.on('messages.upsert', ({ messages, type }) => {
    if (type !== 'notify') return

    for (const msg of messages) {
      if (msg.key.fromMe) continue
      if (!msg.message) continue

      const chatId = msg.key.remoteJid || ''
      const isGroup = chatId.endsWith('@g.us')
      const senderJid = isGroup ? (msg.key.participant || '') : chatId
      const senderName = msg.pushName || ''
      const body = extractBody(msg)
      const hasMedia = hasMediaContent(msg)
      const mediaType = extractMediaType(msg)
      const quotedId = msg.message?.extendedTextMessage?.contextInfo?.stanzaId || ''
      const timestamp = msg.messageTimestamp
        ? Number(msg.messageTimestamp) * 1000
        : Date.now()

      messageQueue.push({
        messageId: msg.key.id || '',
        chatId,
        senderId: senderJid,
        senderName,
        isGroup,
        body,
        hasMedia,
        mediaType,
        mediaUrls: [],
        quotedMessageId: quotedId,
        timestamp,
      })
    }
  })
}

// --- Helpers ---

function extractBody (msg) {
  const m = msg.message
  if (!m) return ''
  return (
    m.conversation ||
    m.extendedTextMessage?.text ||
    m.imageMessage?.caption ||
    m.videoMessage?.caption ||
    m.documentMessage?.caption ||
    ''
  )
}

function hasMediaContent (msg) {
  const m = msg.message
  if (!m) return false
  return !!(m.imageMessage || m.videoMessage || m.audioMessage || m.documentMessage || m.stickerMessage)
}

function extractMediaType (msg) {
  const m = msg.message
  if (!m) return ''
  if (m.imageMessage) return 'image'
  if (m.videoMessage) return 'video'
  if (m.audioMessage) return 'audio'
  if (m.documentMessage) return 'document'
  if (m.stickerMessage) return 'sticker'
  return ''
}

function resolveMediaType (mimeType, fileName) {
  if (!mimeType) {
    const ext = path.extname(fileName || '').toLowerCase()
    const imageExts = ['.jpg', '.jpeg', '.png', '.gif', '.webp']
    const videoExts = ['.mp4', '.mkv', '.mov', '.avi']
    const audioExts = ['.mp3', '.ogg', '.m4a', '.wav', '.opus']
    if (imageExts.includes(ext)) return 'image'
    if (videoExts.includes(ext)) return 'video'
    if (audioExts.includes(ext)) return 'audio'
    return 'document'
  }
  if (mimeType.startsWith('image/')) return 'image'
  if (mimeType.startsWith('video/')) return 'video'
  if (mimeType.startsWith('audio/')) return 'audio'
  return 'document'
}

function normalizeJid (chatId) {
  if (chatId.includes('@')) return chatId
  // asumir numero de telefono
  const digits = chatId.replace(/\D/g, '')
  return digits + '@s.whatsapp.net'
}

function parseArgs (argv) {
  const result = {}
  for (let i = 0; i < argv.length; i += 2) {
    const key = argv[i].replace(/^--/, '')
    result[key] = argv[i + 1]
  }
  return result
}

// --- Main ---
app.listen(PORT, '127.0.0.1', () => {
  logger.info({ port: PORT }, 'http bridge listening')
})

startBaileys().catch((err) => {
  logger.error({ err }, 'baileys start failed')
  process.exit(1)
})
