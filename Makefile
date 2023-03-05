push:
	@git add -A && git commit -m "update" && git push origin master



clean:
	@go clean -modcache



.PHONY: push clean
