up:
	docker-compose up -d


down:
	docker-compose down


clean:
	docker system prune --volumes --force
