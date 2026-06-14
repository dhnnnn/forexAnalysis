const { Client, LocalAuth } = require("whatsapp-web.js");
const qrcode = require("qrcode-terminal");
const express = require("express");
const fs = require("fs");
const path = require("path");

const app = express();
app.use(express.json());

const PORT = process.env.PORT || 3001;
const AUTHORIZED_PHONE = (process.env.WA_TARGET_PHONE || "").replace(/[^0-9]/g, "");
const GO_AGENT_URL = process.env.GO_AGENT_URL || "http://forex-agent:8080";
const SESSION_PATH = "/app/session";

console.log(`🔐 Authorized phone (for alerts): ${AUTHORIZED_PHONE || "(not set)"}`);
console.log(`🔗 Go agent URL: ${GO_AGENT_URL}`);

// ── Cleanup stale Chromium lock files ─────────────────────────────────
function cleanupLockFiles(dir) {
  if (!fs.existsSync(dir)) return;
  const walk = (d) => {
    try {
      const entries = fs.readdirSync(d, { withFileTypes: true });
      for (const entry of entries) {
        const fullPath = path.join(d, entry.name);
        if (entry.isDirectory()) {
          walk(fullPath);
        } else if (entry.name === "SingletonLock" || entry.name === "SingletonCookie" || entry.name === "SingletonSocket") {
          fs.unlinkSync(fullPath);
          console.log(`🧹 Removed lock file: ${fullPath}`);
        }
      }
    } catch (e) { /* ignore */ }
  };
  walk(dir);
}

cleanupLockFiles(SESSION_PATH);

// ── WhatsApp Client Setup ─────────────────────────────────────────────
const client = new Client({
  authStrategy: new LocalAuth({ dataPath: SESSION_PATH }),
  puppeteer: {
    headless: true,
    args: [
      "--no-sandbox",
      "--disable-setuid-sandbox",
      "--disable-dev-shm-usage",
      "--disable-gpu",
      "--single-process",
      "--no-zygote",
      "--disable-extensions",
      "--disable-background-networking",
    ],
  },
});

let isReady = false;

client.on("qr", (qr) => {
  console.log("\n════════════════════════════════════════════════");
  console.log("  📱 Scan QR Code ini dari WhatsApp kamu:");
  console.log("════════════════════════════════════════════════\n");
  qrcode.generate(qr, { small: true });
  console.log("\n════════════════════════════════════════════════");
  console.log("  Buka WhatsApp → Settings → Linked Devices → Link a Device");
  console.log("════════════════════════════════════════════════\n");
});

client.on("ready", () => {
  isReady = true;
  console.log("✅ WhatsApp client connected and ready!");
  console.log(`📞 Listening on port ${PORT}`);
  console.log(`🔐 Authorized phone for alerts: ${AUTHORIZED_PHONE}`);
});

client.on("authenticated", () => {
  console.log("🔑 Authenticated — session saved.");
});

client.on("auth_failure", (msg) => {
  console.error("❌ Auth failure:", msg);
  isReady = false;
});

client.on("disconnected", (reason) => {
  console.log("⚠️ Disconnected:", reason);
  isReady = false;
  setTimeout(() => {
    console.log("🔄 Attempting reconnection...");
    client.initialize();
  }, 5000);
});

// ── Message Listener ──────────────────────────────────────────────────
// WhatsApp sekarang pakai format @lid (Linked ID) bukan @c.us untuk
// beberapa akun. Nomor yang muncul di msg.from bukan nomor telepon asli.
// Harus resolve via contact.number untuk dapat nomor asli.
client.on("message", async (msg) => {
  // Ignore group messages
  if (msg.from.includes("@g.us")) return;

  // Ignore status broadcasts
  if (msg.from === "status@broadcast") return;

  // Ignore pesan dari bot sendiri
  if (msg.fromMe) return;

  // Resolve nomor telepon asli pengirim
  let senderPhone = "";
  try {
    const contact = await msg.getContact();
    // contact.number berisi nomor telepon asli (tanpa +, tanpa spasi)
    senderPhone = (contact.number || "").replace(/[^0-9]/g, "");
  } catch (e) {
    // Fallback: coba extract dari msg.from jika format @c.us
    if (msg.from.endsWith("@c.us")) {
      senderPhone = msg.from.replace("@c.us", "");
    }
  }

  const text = (msg.body || "").trim();
  if (!text) return;

  console.log(`📩 Incoming | from: ${msg.from} | resolved phone: ${senderPhone} | text: "${text}"`);

  // Proceed — bot is privately deployed, respond to all private messages

  try {
    const response = await fetch(`${GO_AGENT_URL}/chat`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        phone: senderPhone || AUTHORIZED_PHONE,
        message: text,
        timestamp: new Date().toISOString(),
      }),
    });

    if (!response.ok) {
      const errText = await response.text();
      console.error(`❌ Go agent error (${response.status}): ${errText}`);
      await msg.reply("⚠️ Bot sedang error, coba lagi nanti.");
      return;
    }

    const data = await response.json();

    if (data.reply) {
      await msg.reply(data.reply);
      console.log(`📤 Replied: "${data.reply.substring(0, 50)}..."`);
    }
  } catch (err) {
    console.error(`❌ Failed to forward to Go agent: ${err.message}`);
    await msg.reply("⚠️ Bot tidak bisa terhubung ke server. Coba lagi nanti.");
  }
});

// ── API Endpoints ─────────────────────────────────────────────────────

app.get("/health", (_req, res) => {
  res.json({
    status: isReady ? "connected" : "disconnected",
    authorizedPhone: AUTHORIZED_PHONE,
    timestamp: new Date().toISOString(),
  });
});

app.post("/send", async (req, res) => {
  try {
    const { phone, message, signal } = req.body;

    if (!phone || !message) {
      return res.status(400).json({ error: "phone and message are required" });
    }

    if (!isReady) {
      return res.status(503).json({ error: "WhatsApp client not ready" });
    }

    const chatId = phone.replace(/[^0-9]/g, "") + "@c.us";
    await client.sendMessage(chatId, message);

    console.log(`📤 Alert sent to ${phone} | Signal: ${signal?.signal || "N/A"}`);

    res.json({ success: true, timestamp: new Date().toISOString(), chatId });
  } catch (err) {
    console.error("❌ Error sending message:", err.message);
    res.status(500).json({ error: err.message });
  }
});

// ── Start Server ──────────────────────────────────────────────────────
app.listen(PORT, "0.0.0.0", () => {
  console.log(`🚀 WhatsApp service starting on port ${PORT}...`);
  console.log("⏳ Initializing WhatsApp client...\n");
  client.initialize();
});
