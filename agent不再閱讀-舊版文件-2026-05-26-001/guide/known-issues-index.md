# ia-analyses-db known issues index

更新日期：2026-05-24-16:27
校準日期：2026-05-24-16:27

- `sync-athena-*` 雖有 Makefile 正式入口，但本輪未補完整 source-level 入口索引；已記入 `guide/code-navigation-backlog.md`
- `salespipe` 與 `sales` / `validation` 邊界目前主要靠型別與檔名導航，尚未補短導航註記
- `文件/過時/` 內保留大量 phase2c 歷史文件；新工作若需引用，應先回到 active 文件與 guide 路由
- clone 後若本機沒有 `backup/dev/baseline/*.dump`，`make dev-restore-baseline` 仍會明確失敗；目前 baseline handoff 仍需 out-of-band 傳遞 local dump