TIMEOUT=30m



test-full:
	go test -v -count=1 -timeout $(TIMEOUT) -run '^Test_Run_Full_Analysis$$' ./tests/...

test-standard:
	go test -v -count=1 -timeout $(TIMEOUT) -run '^Test_RunAll_Standard$$' ./tests/...

test-ddos:
	go test -v -count=1 -timeout $(TIMEOUT) -run '^Test_RunAll_DDoS$$' ./tests/...

t-all: test-full
