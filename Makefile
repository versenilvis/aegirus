TIMEOUT=30m


test-full:
	go test -v -timeout $(TIMEOUT) -run '^Test_Run_Full_Analysis$$' ./tests/...

test-standard:
	go test -v -timeout $(TIMEOUT) -run '^Test_RunAll_Standard$$' ./tests/...

test-ddos:
	go test -v -timeout $(TIMEOUT) -run '^Test_RunAll_DDoS$$' ./tests/...

t-all: test-full
