test:
	bash -c 'rm -f coverage.txt && cd src && go test -coverprofile=gpbt.cover.out -covermode=atomic && cat *.cover.out >> ../coverage.txt'
coverage:
	bash -c 'make test'
	bash -c 'cd src && go tool cover -html=gpbt.cover.out -o ../coverage.html'
build:
	bash -c 'cd src && go build ./...'
install:
	glide install