# General Commands

gofmtcheck:
	need_fmt=$$(gofmt -l $$(find . -type f -name '*.go' -not -path './vendor/*'));\
	if [ "$$need_fmt" = "" ]; then echo "hooray"; else echo "files that need formatting:"; echo $$need_fmt; exit 1; fi

startdev:
	cd clients && npm start