PROJECT_NAME := pandemic-api
PKG := github.com/gidyon/koviapp-ussd

setup_dev: ## start development databases
	cd deployments/compose && docker-compose up -d

teardown_dev: ## stop development databases
	cd deployments/compose && docker-compose down

compile:
	go build -i -v -o ussd $(PKG)/cmd

run:
	./ussd -config-file=configs/config.dev.yml
	
docker_build:
ifdef tag
	@docker build -t gidyon/$(PROJECT_NAME)-ussd:$(tag) .
else
	@docker build -t gidyon/$(PROJECT_NAME)-ussd:latest .
endif

docker_tag:
ifdef tag
	@docker tag gidyon/$(PROJECT_NAME)-ussd:$(tag) gidyon/$(PROJECT_NAME)-ussd:$(tag)
else
	@docker tag gidyon/$(PROJECT_NAME)-ussd:latest gidyon/$(PROJECT_NAME)-ussd:latest
endif

docker_push:
ifdef tag
	@docker push gidyon/$(PROJECT_NAME)-ussd:$(tag)
else
	@docker push gidyon/$(PROJECT_NAME)-ussd:latest
endif

build_image: docker_build docker_tag docker_push

build: compile docker_build docker_tag docker_push