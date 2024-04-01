CONSUL_CE_IMAGE = hashicorp/consul
CONSUL_CE_VERSIONS = 1.18 1.17 1.16 1.15

CONSUL_ENT_IMAGE = hashicorp/consul-enterprise
CONSUL_ENT_VERSIONS = 1.17-ent 1.16-ent 1.15-ent

CONSUL_TEMPLATE_TAG = v0.34.0

build-ce-images:
	@echo "==> running build-ce-images"
	for version in $(CONSUL_CE_VERSIONS); do \
		docker build -t ct-consul-compat-$$version \
		 --build-arg CONSUL_IMAGE_VERSION=$$version \
		 --build-arg CONSUL_IMAGE_NAME=${CONSUL_CE_IMAGE} \
		 --build-arg CONSUL_TEMPLATE_TAG=${CONSUL_TEMPLATE_TAG} \
		 --no-cache .; \
	done

.PHONY: .build-ce-images

build-ent-images:
	@echo "==> running build-ent-images"
	for version in $(CONSUL_ENT_VERSIONS); do \
		docker build -t ct-consul-compat-$$version \
		 --build-arg CONSUL_IMAGE_VERSION=$$version \
		 --build-arg CONSUL_IMAGE_NAME=${CONSUL_ENT_IMAGE} \
		 --build-arg CONSUL_TEMPLATE_TAG=${CONSUL_TEMPLATE_TAG} \
		 --no-cache .; \
	done

.PHONY: .build-ent-images

test-compat-ce:
	@echo "==> running test-compat-ce"
	for version in $(CONSUL_CE_VERSIONS); do \
		docker run --privileged --init ct-consul-compat-$$version; \
	done

.PHONY: test-compat-ce

test-compat-ent:
	@echo "==> running test-compat-ent"
	for version in $(CONSUL_ENT_VERSIONS); do \
		docker run --privileged -e CONSUL_LICENSE=$(CONSUL_LICENSE) --init ct-consul-compat-$$version; \
	done

.PHONY: test-compat-ent

test-compat:
	@echo "==> running run"
	build-ce-images
	test-compat-ce
	build-ent-images
	test-compat-ent
	done

.PHONY: test-compat

clean:
	@echo "==> running clean"
	docker system prune -y

.PHONY: clean
