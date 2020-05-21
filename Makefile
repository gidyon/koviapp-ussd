PROJECT_NAME := pandemic-api
PKG := gituhub.com/gidyon/koviapp-ussd

compile:
	go build -i -v -o ussd $(PKG)/cmd

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