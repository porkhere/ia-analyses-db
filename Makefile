SHELL := /bin/bash

PROJECT_DIR := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
ENV_FILE := $(PROJECT_DIR).env
DEV_ENV_SOURCE := $(PROJECT_DIR).env.dev
PROD_ENV_SOURCE := $(PROJECT_DIR).env.prod
DOCKER_COMPOSE := docker compose --project-directory $(PROJECT_DIR) --env-file $(ENV_FILE)

.DEFAULT_GOAL := help

.PHONY: help env-status \
	dev-env prod-env \
	dev-up dev-wait dev-down dev-restart dev-migrate dev-size dev-backup dev-backup-list dev-sync-seeds dev-apply-patches dev-del-backup dev-restore dev-mark-runtime-aligned dev-restore-baseline dev-smoke-analytics \
	prod-up prod-wait prod-down prod-restart prod-migrate prod-size prod-backup prod-backup-list prod-sync-seeds prod-apply-patches prod-del-backup prod-restore prod-mark-runtime-aligned \
	sales-pipe-status sales-pipe-plan sales-pipe-validate sales-pipe-write-local sales-pipe-resume sales-pipe-report \
	sync-athena-dry sync-athena-dry-fast sync-athena-dry-full sync-athena-validate sync-athena-write-plan sync-athena-write-local \
	sync-sales-dims-plan sync-sales-dims \
	db-up db-wait db-down db-restart db-size db-backup db-backup-list db-sync-seeds db-apply-patches db-del-backup db-restore db-sync-athena

define require_env
	@if [ ! -f "$(ENV_FILE)" ]; then echo "找不到 .env，請先執行 make $(1)-env"; exit 1; fi
	@if ! grep -Eq '^APP_ENV=$(1)$$' "$(ENV_FILE)"; then echo ".env 目前不是 $(1) 環境，請先執行 make $(1)-env"; exit 1; fi
endef

define require_current_env
	@if [ ! -f "$(ENV_FILE)" ]; then echo "找不到 .env，請先執行 make dev-env 或 make prod-env"; exit 1; fi
endef

define install_env
	@if [ ! -f "$(2)" ]; then echo "找不到環境檔：$(2)"; exit 1; fi
	@cp "$(2)" "$(ENV_FILE)"
	@echo ".env 已切換為 $(1) 環境"
endef

help:
	@echo "ia-analyses-db 操作入口"
	@echo ""
	@echo "環境切換"
	@echo "  make dev-env                  以 .env.dev 產生目前工作 .env"
	@echo "  make prod-env                 以 .env.prod 產生目前工作 .env"
	@echo ""
	@echo "容器與資料庫"
	@echo "  make dev-up [RESTORE=1] [BACKUP_FILE=YYYY-MM-DD-HH-MM.dump]"
	@echo "  make dev-down"
	@echo "  make dev-restart"
	@echo "  make dev-migrate"
	@echo "  make dev-size"
	@echo "  make dev-backup"
	@echo "  make dev-backup-list"
	@echo "  make dev-restore [BACKUP_FILE=YYYY-MM-DD-HH-MM.dump]"
	@echo "  make dev-mark-runtime-aligned BACKUP_FILE=YYYY-MM-DD-HH-MM.dump"
	@echo "  make dev-restore-baseline [BASELINE_FILE=YYYY-MM-DD-HH-MM.dump]"
	@echo "  make dev-smoke-analytics"
	@echo "  make prod-up [RESTORE=1] [BACKUP_FILE=YYYY-MM-DD-HH-MM.dump]"
	@echo "  make prod-down"
	@echo "  make prod-restart"
	@echo "  make prod-migrate"
	@echo "  make prod-size"
	@echo "  make prod-backup"
	@echo "  make prod-backup-list"
	@echo "  make prod-restore [BACKUP_FILE=YYYY-MM-DD-HH-MM.dump]"
	@echo "  make prod-mark-runtime-aligned BACKUP_FILE=YYYY-MM-DD-HH-MM.dump"
	@echo ""
	@echo "維護工具"
	@echo "  make dev-sync-seeds          依目前 schema 檔重跑 seed 同步"
	@echo "  make dev-apply-patches       套用 db/patches/*.sql"
	@echo "  make dev-del-backup [BACKUP_FILE=YYYY-MM-DD-HH-MM.dump|ALL=1]"
	@echo "  make prod-sync-seeds"
	@echo "  make prod-apply-patches"
	@echo "  make prod-del-backup [BACKUP_FILE=YYYY-MM-DD-HH-MM.dump|ALL=1]"
	@echo ""
	@echo "分析流程"
	@echo "  make sales-pipe-status"
	@echo "  make sales-pipe-plan OWNER_USER_KEY=<key> OWNER_USER_ID=<id> START_DATE=YYYY-MM-DD END_DATE=YYYY-MM-DD"
	@echo "  make sales-pipe-validate OWNER_USER_KEY=<key> OWNER_USER_ID=<id> START_DATE=YYYY-MM-DD END_DATE=YYYY-MM-DD"
	@echo "  make sales-pipe-write-local OWNER_USER_KEY=<key> OWNER_USER_ID=<id> START_DATE=YYYY-MM-DD END_DATE=YYYY-MM-DD [CONFIRM_LONG_RUN=1]"
	@echo "  make sales-pipe-resume [FORCE=1] [CONFIRM_LONG_RUN=1]"
	@echo "  make sales-pipe-report"
	@echo "  make sync-athena-dry-fast OWNER_USER_KEY=<key> START_DATE=YYYY-MM-DD [END_DATE=YYYY-MM-DD] [PREVIEW_LIMIT=20]"
	@echo "  make sync-athena-dry-full OWNER_USER_KEY=<key> START_DATE=YYYY-MM-DD [END_DATE=YYYY-MM-DD] [PREVIEW_LIMIT=20]"
	@echo "  make sync-athena-validate OWNER_USER_KEY=<key> OWNER_USER_ID=<id> START_DATE=YYYY-MM-DD [END_DATE=YYYY-MM-DD]"
	@echo "  make sync-athena-write-plan OWNER_USER_KEY=<key> OWNER_USER_ID=<id> START_DATE=YYYY-MM-DD [END_DATE=YYYY-MM-DD]"
	@echo "  make sync-athena-write-local OWNER_USER_KEY=<key> OWNER_USER_ID=<id> START_DATE=YYYY-MM-DD END_DATE=YYYY-MM-DD  # 最多 2 天"
	@echo "  make sync-sales-dims-plan OWNER_USER_KEY=<key> OWNER_USER_ID=<id> START_DATE=YYYY-MM-DD [END_DATE=YYYY-MM-DD]"
	@echo "  make sync-sales-dims OWNER_USER_KEY=<key> OWNER_USER_ID=<id> START_DATE=YYYY-MM-DD [END_DATE=YYYY-MM-DD]"
	@echo ""
	@echo "說明"
	@echo "  - 所有 sales-pipe-*、sync-athena-*、sync-sales-dims* 都會使用目前 .env"
	@echo "  - 切環境前請先執行 make dev-env 或 make prod-env"
	@echo "  - 舊 db-* 命名已退役，正式入口以 dev-* / prod-* 為準"

env-status:
	@$(call require_current_env)
	@grep '^APP_ENV=' "$(ENV_FILE)" | cut -d= -f2 | xargs -I{} echo "current_app_env: {}"

dev-env:
	$(call install_env,dev,$(DEV_ENV_SOURCE))

prod-env:
	$(call install_env,prod,$(PROD_ENV_SOURCE))

dev-up:
	$(call require_env,dev)
	@$(DOCKER_COMPOSE) up -d postgres adminer
	@$(MAKE) --no-print-directory dev-wait
	@if [ "$(RESTORE)" = "1" ]; then $(MAKE) --no-print-directory dev-restore BACKUP_FILE="$(BACKUP_FILE)"; fi
	@$(MAKE) --no-print-directory dev-migrate

dev-wait:
	$(call require_env,dev)
	@$(DOCKER_COMPOSE) exec -T postgres sh -lc 'until pg_isready -U "$$POSTGRES_USER" -d "$$POSTGRES_DB" >/dev/null 2>&1; do sleep 1; done'

dev-down:
	$(call require_env,dev)
	@$(DOCKER_COMPOSE) down

dev-restart:
	$(call require_env,dev)
	@$(MAKE) --no-print-directory dev-down
	@$(MAKE) --no-print-directory dev-up RESTORE="$(RESTORE)" BACKUP_FILE="$(BACKUP_FILE)"

dev-migrate:
	$(call require_env,dev)
	@$(PROJECT_DIR)scripts/db_apply_patches.sh

dev-size:
	$(call require_env,dev)
	@$(PROJECT_DIR)scripts/db_size.sh

dev-backup:
	$(call require_env,dev)
	@$(PROJECT_DIR)scripts/db_backup.sh

dev-backup-list:
	$(call require_env,dev)
	@$(PROJECT_DIR)scripts/db_backup_list.sh

dev-sync-seeds:
	$(call require_env,dev)
	@$(PROJECT_DIR)scripts/db_sync_seeds.sh

dev-apply-patches:
	$(call require_env,dev)
	@$(PROJECT_DIR)scripts/db_apply_patches.sh

dev-del-backup:
	$(call require_env,dev)
	@if [ "$(ALL)" = "1" ]; then $(PROJECT_DIR)scripts/db_del_backup.sh all; else $(PROJECT_DIR)scripts/db_del_backup.sh "$(BACKUP_FILE)"; fi

dev-restore:
	$(call require_env,dev)
	@$(PROJECT_DIR)scripts/db_restore.sh "$(BACKUP_FILE)"

dev-mark-runtime-aligned:
	$(call require_env,dev)
	@if [ -z "$(BACKUP_FILE)" ]; then echo "BACKUP_FILE 為必填"; exit 1; fi
	@$(PROJECT_DIR)scripts/db_mark_runtime_aligned.sh "$(BACKUP_FILE)"

dev-restore-baseline:
	$(call require_env,dev)
	@$(PROJECT_DIR)scripts/db_restore_baseline.sh "$(BASELINE_FILE)"

dev-smoke-analytics:
	$(call require_env,dev)
	@$(PROJECT_DIR)scripts/db_smoke_analytics.sh

prod-up:
	$(call require_env,prod)
	@$(DOCKER_COMPOSE) up -d postgres adminer
	@$(MAKE) --no-print-directory prod-wait
	@if [ "$(RESTORE)" = "1" ]; then $(MAKE) --no-print-directory prod-restore BACKUP_FILE="$(BACKUP_FILE)"; fi
	@$(MAKE) --no-print-directory prod-migrate

prod-wait:
	$(call require_env,prod)
	@$(DOCKER_COMPOSE) exec -T postgres sh -lc 'until pg_isready -U "$$POSTGRES_USER" -d "$$POSTGRES_DB" >/dev/null 2>&1; do sleep 1; done'

prod-down:
	$(call require_env,prod)
	@$(DOCKER_COMPOSE) down

prod-restart:
	$(call require_env,prod)
	@$(MAKE) --no-print-directory prod-down
	@$(MAKE) --no-print-directory prod-up RESTORE="$(RESTORE)" BACKUP_FILE="$(BACKUP_FILE)"

prod-migrate:
	$(call require_env,prod)
	@$(PROJECT_DIR)scripts/db_apply_patches.sh

prod-size:
	$(call require_env,prod)
	@$(PROJECT_DIR)scripts/db_size.sh

prod-backup:
	$(call require_env,prod)
	@$(PROJECT_DIR)scripts/db_backup.sh

prod-backup-list:
	$(call require_env,prod)
	@$(PROJECT_DIR)scripts/db_backup_list.sh

prod-sync-seeds:
	$(call require_env,prod)
	@$(PROJECT_DIR)scripts/db_sync_seeds.sh

prod-apply-patches:
	$(call require_env,prod)
	@$(PROJECT_DIR)scripts/db_apply_patches.sh

prod-del-backup:
	$(call require_env,prod)
	@if [ "$(ALL)" = "1" ]; then $(PROJECT_DIR)scripts/db_del_backup.sh all; else $(PROJECT_DIR)scripts/db_del_backup.sh "$(BACKUP_FILE)"; fi

prod-restore:
	$(call require_env,prod)
	@$(PROJECT_DIR)scripts/db_restore.sh "$(BACKUP_FILE)"

prod-mark-runtime-aligned:
	$(call require_env,prod)
	@if [ -z "$(BACKUP_FILE)" ]; then echo "BACKUP_FILE 為必填"; exit 1; fi
	@$(PROJECT_DIR)scripts/db_mark_runtime_aligned.sh "$(BACKUP_FILE)"

sales-pipe-status:
	@$(call require_current_env)
	@set -a; \
	  . "$(ENV_FILE)"; \
	  set +a; \
	  cd "$(PROJECT_DIR)" && go run ./cmd/sales-pipe --mode status

sales-pipe-plan:
	@$(call require_current_env)
	@if [ -z "$(OWNER_USER_KEY)" ] || [ -z "$(OWNER_USER_ID)" ] || [ -z "$(START_DATE)" ] || [ -z "$(END_DATE)" ]; then echo "OWNER_USER_KEY、OWNER_USER_ID、START_DATE 與 END_DATE 為必填"; exit 1; fi
	@set -a; \
	  . "$(ENV_FILE)"; \
	  set +a; \
	  cd "$(PROJECT_DIR)" && go run ./cmd/sales-pipe --mode write-plan --owner-user-key "$(OWNER_USER_KEY)" --owner-user-id "$(OWNER_USER_ID)" --start-date "$(START_DATE)" --end-date "$(END_DATE)"

sales-pipe-validate:
	@$(call require_current_env)
	@if [ -z "$(OWNER_USER_KEY)" ] || [ -z "$(OWNER_USER_ID)" ] || [ -z "$(START_DATE)" ] || [ -z "$(END_DATE)" ]; then echo "OWNER_USER_KEY、OWNER_USER_ID、START_DATE 與 END_DATE 為必填"; exit 1; fi
	@set -a; \
	  . "$(ENV_FILE)"; \
	  set +a; \
	  cd "$(PROJECT_DIR)" && go run ./cmd/sales-pipe --mode validate-only --owner-user-key "$(OWNER_USER_KEY)" --owner-user-id "$(OWNER_USER_ID)" --start-date "$(START_DATE)" --end-date "$(END_DATE)"

sales-pipe-write-local:
	@$(call require_current_env)
	@if [ -z "$(OWNER_USER_KEY)" ] || [ -z "$(OWNER_USER_ID)" ] || [ -z "$(START_DATE)" ] || [ -z "$(END_DATE)" ]; then echo "OWNER_USER_KEY、OWNER_USER_ID、START_DATE 與 END_DATE 為必填"; exit 1; fi
	@set -a; \
	  . "$(ENV_FILE)"; \
	  set +a; \
	  cd "$(PROJECT_DIR)" && go run ./cmd/sales-pipe --mode write-local --owner-user-key "$(OWNER_USER_KEY)" --owner-user-id "$(OWNER_USER_ID)" --start-date "$(START_DATE)" --end-date "$(END_DATE)" $(if $(filter 1,$(CONFIRM_LONG_RUN)),--confirm-long-run,) $(if $(filter 1,$(FORCE)),--force,)

sales-pipe-resume:
	@$(call require_current_env)
	@set -a; \
	  . "$(ENV_FILE)"; \
	  set +a; \
	  cd "$(PROJECT_DIR)" && go run ./cmd/sales-pipe --mode resume $(if $(OWNER_USER_KEY),--owner-user-key "$(OWNER_USER_KEY)",) $(if $(OWNER_USER_ID),--owner-user-id "$(OWNER_USER_ID)",) $(if $(START_DATE),--start-date "$(START_DATE)",) $(if $(END_DATE),--end-date "$(END_DATE)",) $(if $(filter 1,$(CONFIRM_LONG_RUN)),--confirm-long-run,) $(if $(filter 1,$(FORCE)),--force,)

sales-pipe-report:
	@$(call require_current_env)
	@set -a; \
	  . "$(ENV_FILE)"; \
	  set +a; \
	  cd "$(PROJECT_DIR)" && go run ./cmd/sales-pipe --mode report

sync-athena-dry: sync-athena-dry-fast

sync-athena-dry-fast:
	@$(call require_current_env)
	@if [ -z "$(OWNER_USER_KEY)" ] || [ -z "$(START_DATE)" ]; then echo "OWNER_USER_KEY 與 START_DATE 為必填"; exit 1; fi
	@set -a; \
	  . "$(ENV_FILE)"; \
	  set +a; \
	  cd "$(PROJECT_DIR)" && go run ./cmd/sync-athena --owner-user-key "$(OWNER_USER_KEY)" --start-date "$(START_DATE)" $(if $(END_DATE),--end-date "$(END_DATE)",) $(if $(PREVIEW_LIMIT),--preview-limit "$(PREVIEW_LIMIT)",) --dry-run --dry-run-mode fast

sync-athena-dry-full:
	@$(call require_current_env)
	@if [ -z "$(OWNER_USER_KEY)" ] || [ -z "$(START_DATE)" ]; then echo "OWNER_USER_KEY 與 START_DATE 為必填"; exit 1; fi
	@set -a; \
	  . "$(ENV_FILE)"; \
	  set +a; \
	  cd "$(PROJECT_DIR)" && go run ./cmd/sync-athena --owner-user-key "$(OWNER_USER_KEY)" --start-date "$(START_DATE)" $(if $(END_DATE),--end-date "$(END_DATE)",) $(if $(PREVIEW_LIMIT),--preview-limit "$(PREVIEW_LIMIT)",) --dry-run --dry-run-mode full

sync-athena-validate:
	@$(call require_current_env)
	@if [ -z "$(OWNER_USER_KEY)" ] || [ -z "$(OWNER_USER_ID)" ] || [ -z "$(START_DATE)" ]; then echo "OWNER_USER_KEY、OWNER_USER_ID 與 START_DATE 為必填"; exit 1; fi
	@set -a; \
	  . "$(ENV_FILE)"; \
	  set +a; \
	  cd "$(PROJECT_DIR)" && go run ./cmd/sync-athena --owner-user-key "$(OWNER_USER_KEY)" --owner-user-id "$(OWNER_USER_ID)" --start-date "$(START_DATE)" $(if $(END_DATE),--end-date "$(END_DATE)",) --validate-only

sync-athena-write-plan:
	@$(call require_current_env)
	@if [ -z "$(OWNER_USER_KEY)" ] || [ -z "$(OWNER_USER_ID)" ] || [ -z "$(START_DATE)" ]; then echo "OWNER_USER_KEY、OWNER_USER_ID 與 START_DATE 為必填"; exit 1; fi
	@set -a; \
	  . "$(ENV_FILE)"; \
	  set +a; \
	  cd "$(PROJECT_DIR)" && go run ./cmd/sync-athena --owner-user-key "$(OWNER_USER_KEY)" --owner-user-id "$(OWNER_USER_ID)" --start-date "$(START_DATE)" $(if $(END_DATE),--end-date "$(END_DATE)",) --write-pg

sync-athena-write-local:
	@$(call require_current_env)
	@if [ -z "$(OWNER_USER_KEY)" ] || [ -z "$(OWNER_USER_ID)" ] || [ -z "$(START_DATE)" ] || [ -z "$(END_DATE)" ]; then echo "OWNER_USER_KEY、OWNER_USER_ID、START_DATE 與 END_DATE 為必填"; exit 1; fi
	@set -a; \
	  . "$(ENV_FILE)"; \
	  set +a; \
	  cd "$(PROJECT_DIR)" && go run ./cmd/sync-athena --owner-user-key "$(OWNER_USER_KEY)" --owner-user-id "$(OWNER_USER_ID)" --start-date "$(START_DATE)" --end-date "$(END_DATE)" --write-pg --local-only-actual-write

sync-sales-dims-plan:
	@$(call require_current_env)
	@if [ -z "$(OWNER_USER_KEY)" ] || [ -z "$(OWNER_USER_ID)" ] || [ -z "$(START_DATE)" ]; then echo "OWNER_USER_KEY、OWNER_USER_ID 與 START_DATE 為必填"; exit 1; fi
	@set -a; \
	  . "$(ENV_FILE)"; \
	  set +a; \
	  cd "$(PROJECT_DIR)" && go run ./cmd/sync-sales-dims --owner-user-key "$(OWNER_USER_KEY)" --owner-user-id "$(OWNER_USER_ID)" --start-date "$(START_DATE)" $(if $(END_DATE),--end-date "$(END_DATE)",) --plan

sync-sales-dims:
	@$(call require_current_env)
	@if [ -z "$(OWNER_USER_KEY)" ] || [ -z "$(OWNER_USER_ID)" ] || [ -z "$(START_DATE)" ]; then echo "OWNER_USER_KEY、OWNER_USER_ID 與 START_DATE 為必填"; exit 1; fi
	@set -a; \
	  . "$(ENV_FILE)"; \
	  set +a; \
	  cd "$(PROJECT_DIR)" && go run ./cmd/sync-sales-dims --owner-user-key "$(OWNER_USER_KEY)" --owner-user-id "$(OWNER_USER_ID)" --start-date "$(START_DATE)" $(if $(END_DATE),--end-date "$(END_DATE)",) --apply

db-up:
	@echo "db-up 已退役；請改用 make dev-up 或 make prod-up"
	@exit 1

db-wait:
	@echo "db-wait 已退役；請改用 make dev-wait 或 make prod-wait"
	@exit 1

db-down:
	@echo "db-down 已退役；請改用 make dev-down 或 make prod-down"
	@exit 1

db-restart:
	@echo "db-restart 已退役；請改用 make dev-restart 或 make prod-restart"
	@exit 1

db-size:
	@echo "db-size 已退役；請改用 make dev-size 或 make prod-size"
	@exit 1

db-backup:
	@echo "db-backup 已退役；請改用 make dev-backup 或 make prod-backup"
	@exit 1

db-backup-list:
	@echo "db-backup-list 已退役；請改用 make dev-backup-list 或 make prod-backup-list"
	@exit 1

db-sync-seeds:
	@echo "db-sync-seeds 已退役；請改用 make dev-sync-seeds 或 make prod-sync-seeds"
	@exit 1

db-apply-patches:
	@echo "db-apply-patches 已退役；請改用 make dev-apply-patches 或 make prod-apply-patches"
	@exit 1

db-del-backup:
	@echo "db-del-backup 已退役；請改用 make dev-del-backup 或 make prod-del-backup"
	@exit 1

db-restore:
	@echo "db-restore 已退役；請改用 make dev-restore 或 make prod-restore"
	@exit 1

db-sync-athena:
	@echo "db-sync-athena 已停用；請先執行 make dev-env 或 make prod-env，然後改用 sales-pipe-* 或 sync-athena-* / sync-sales-dims*"
	@exit 1