.PHONY: docs setup frontend-setup backend-setup dev-data
.PHONY: fixture/main.transactions.json

DEV_CONFIG := $(CURDIR)/dev_data/paisa.yaml
DEV_PORT := 7501

setup: frontend-setup backend-setup

frontend-setup:
	npm install
	npm run build

backend-setup:
	go mod download

develop:
	./node_modules/.bin/concurrently --names "GO,JS" -c "auto" "make serve" "npm run dev"

# Generate a dev config (once) pointing the journal at dev_data/Ledgers/main.ledger.
$(DEV_CONFIG):
	printf 'journal_path: Ledgers/main.ledger\ndb_path: paisa.db\n' > $@

# Start the dev server (Go API + frontend) using the dev_data ledgers.
dev-data: $(DEV_CONFIG)
	PAISA_CONFIG=$(DEV_CONFIG) go run . update
	PAISA_CONFIG=$(DEV_CONFIG) PAISA_PORT=$(DEV_PORT) ./node_modules/.bin/concurrently --names "GO,JS" -c "auto" "make serve PORT=$(DEV_PORT)" "npm run dev"

PORT ?= 7500
serve:
	./node_modules/.bin/nodemon --signal SIGTERM --delay 2000ms --watch '.' --ext go,json --exec 'go run . serve -p $(PORT) || exit 1'

debug:
	./node_modules/.bin/concurrently --names "GO,JS" -c "auto" "make serve-now" "npm run dev"

serve-now:
	./node_modules/.bin/nodemon --signal SIGTERM --delay 2000ms --watch '.' --ext go,json --exec 'TZ=UTC go run . serve --now 2022-02-07 || exit 1'


watch:
	npm run "build:watch"
docs:
	mkdocs serve -a 0.0.0.0:8000

sample:
	go build && ./paisa init && ./paisa update

publish:
	nix develop --command bash -c 'mkdocs build'

parser:
	npm run parser-build-debug

lint:
	./node_modules/.bin/prettier --check src
	npm run check
	test -z $$(gofmt -l .)

regen:
	go build
	unset PAISA_CONFIG && REGENERATE=true TZ=UTC bun test tests

jstest:
	bun test --preload ./src/happydom.ts src
	go build
	unset PAISA_CONFIG && TZ=UTC bun test tests

jsbuild:
	npm run build

test: jsbuild jstest
	go test ./...

windows:
	GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CXX=x86_64-w64-mingw32-g++ CC=x86_64-w64-mingw32-gcc go build


deploy:
	fly scale count 2 --region lax --yes
	docker build -t paisa . --file Dockerfile.demo
	fly deploy -i paisa:latest --local-only
	fly scale count 1 --region lax --yes

install:
	npm run build
	go build
	go install

fixture/main.transactions.json:
	cd /tmp && paisa init
	cp fixture/main.ledger /tmp/main.ledger
	cd /tmp && paisa update --journal && paisa serve -p 6500 &
	sleep 1
	curl http://localhost:6500/api/transaction | jq .transactions > fixture/main.transactions.json
	pkill -f 'paisa serve -p 6500'

generate-fonts:
	bun download-svgs.js
	node generate-font.js

# manually remove canvas (optional) package as it's broken on nix
node2nix:
	npm install --lockfile-version 2
	node2nix --development -18 --input package.json \
	--lock package-lock.json \
	--node-env ./flake/node-env.nix \
	--composition ./flake/default.nix \
	--output ./flake/node-package.nix
