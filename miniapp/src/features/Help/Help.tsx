import { useTranslation } from "react-i18next"

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card"

/** Справка: что умеет мини-апп и какие команды остались у бота. */
export function Help() {
  const { t } = useTranslation()

  const commands = [
    { cmd: "/start", key: "help.cmdStart" },
    { cmd: "/find", key: "help.cmdFind" },
    { cmd: "/list", key: "help.cmdList" },
    { cmd: "/help", key: "help.cmdHelp" },
  ] as const

  return (
    <section className="space-y-3">
      <h1 className="text-lg font-semibold">{t("help.title")}</h1>
      <Card>
        <CardContent className="space-y-3 pt-6 text-sm">
          <p>{t("help.intro")}</p>
          <p className="text-muted-foreground">{t("help.notifications")}</p>
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("help.commandsTitle")}</CardTitle>
        </CardHeader>
        <CardContent>
          <dl className="space-y-2 text-sm">
            {commands.map(({ cmd, key }) => (
              <div key={cmd} className="flex gap-3">
                <dt className="shrink-0 font-mono text-muted-foreground">{cmd}</dt>
                <dd>{t(key)}</dd>
              </div>
            ))}
          </dl>
        </CardContent>
      </Card>
    </section>
  )
}
