/**
 * Dev-инструмент: вычисляет валидный initData-квиток, подписанный бот-токеном,
 * и записывает его в .env.local (VITE_DEV_INIT_DATA). Без него mock-квиток
 * из telegram.ts подписан фиктивным hash, и miniapi отвечает 401.
 *
 * Токен читается из ../backend/.env и остаётся там — во фронт и git он не
 * попадает (.env.local в .gitignore). Квиток живёт MINIAPI_INITDATA_MAX_AGE
 * (24 ч по умолчанию) — перезапускайте скрипт, когда экраны начнут ловить 401.
 *
 * Запуск: npm run dev:auth
 */
import { createHmac } from "node:crypto"
import { readFileSync, writeFileSync } from "node:fs"
import { dirname, join } from "node:path"
import { fileURLToPath } from "node:url"

const here = dirname(fileURLToPath(import.meta.url))
const backendEnvPath = join(here, "..", "..", "backend", ".env")
const envLocalPath = join(here, "..", ".env.local")

function readBotToken() {
  let raw
  try {
    raw = readFileSync(backendEnvPath, "utf8")
  } catch {
    console.error(
      `Не найден ${backendEnvPath} — запустите из клона репозитория с настроенным backend/.env`,
    )
    process.exit(1)
  }
  const line = raw.split(/\r?\n/).find((l) => l.startsWith("TELEGRAM_BOT_TOKEN="))
  const token = line?.slice("TELEGRAM_BOT_TOKEN=".length).trim()
  if (!token || token === "your_bot_token_here") {
    console.error("В backend/.env нет реального TELEGRAM_BOT_TOKEN")
    process.exit(1)
  }
  return token
}

/** Тот же пользователь, что и в mock telegram.ts (см. mockInitData). */
const user = JSON.stringify({
  id: 777_000_001,
  first_name: "Dev",
  last_name: "User",
  username: "dev_user",
  language_code: "ru",
})
const authDate = Math.floor(Date.now() / 1000)
// signature сервер (miniapi) не проверяет, но схема SDK в mockTelegramEnv
// требует это поле — без него dev-квиток роняет инициализацию приложения.
const signature = "dev-signature"

// data-check-string: все поля, кроме hash, по алфавиту (auth_date < signature < user)
const dataCheckString = `auth_date=${authDate}\nsignature=${signature}\nuser=${user}`
const secret = createHmac("sha256", "WebAppData").update(readBotToken()).digest()
const hash = createHmac("sha256", secret).update(dataCheckString).digest("hex")

const initData =
  `auth_date=${authDate}&signature=${signature}` + `&user=${encodeURIComponent(user)}&hash=${hash}`
writeFileSync(envLocalPath, `VITE_DEV_INIT_DATA=${initData}\n`)

console.log("Готово: .env.local записан, квиток действителен 24 ч.")
console.log("Перезапустите dev-сервер (npm run dev), чтобы Vite подхватил переменную.")
