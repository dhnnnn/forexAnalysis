const { Client, LocalAuth } = require("whatsapp-web.js");
const qrcode = require("qrcode-terminal");
const express = require("express");

const app = express();
app.use(express.json());

const PORT = process.env.PORT || 3001;

// ── WhatsApp Client Setup ─────────────────────────────────────────────
const client = new Client({
  authStrategy: new LocalAuth({ dataPath: "/app/session" }),
  puppeteer: {
    headless: true,
    args: [
      "--no-sandbox",
      "--disable-setuid-sandbox",
      "--disable-dev-shm-usage",
      "--disable-gpu",
      "--single-process",
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
  // Auto-reconnect after 5 seconds
  setTimeout(() => {
    console.log("🔄 Attempting reconnection...");
    client.initialize();
  }, 5000);
});

// ── API Endpoints ─────────────────────────────────────────────────────

// Health check
app.get("/health", (req, res) => {
  res.json({
    status: isReady ? "connected" : "disconnected",
    timestamp: new Date().toISOString(),
  });
});

// Send message endpoint (called by Go agent)
app.post("/send", async (req, res) => {
  try {
    const { phone, message, signal } = req.body;

    if (!phone || !message) {
      return res.status(400).json({ error: "phone and message are required" });
    }

    if (!isReady) {
      return res.status(503).json({ error: "WhatsApp client not ready — scan QR first" });
    }

    // Format phone number: remove +, add @c.us suffix
    const chatId = phone.replace(/[^0-9]/g, "") + "@c.us";

    await client.sendMessage(chatId, message);

    console.log(`📤 Message sent to ${phone} | Signal: ${signal?.signal || "N/A"} | Pair: ${signal?.pair || "N/A"}`);

    res.json({
      success: true,
      timestamp: new Date().toISOString(),
      chatId,
    });
  } catch (err) {
    console.error("❌ Error sending message:", err.message);
    res.status(500).json({ error: err.message });
  }
});

// ── Start Server ──────────────────────────────────────────────────────
app.listen(PORT, "0.0.0.0", () => {
  console.log(`🚀 WhatsApp service starting on port ${PORT}...`);
  console.log("⏳ Initializing WhatsApp client (wait for QR code)...\n");
  client.initialize();
});
