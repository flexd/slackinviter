.PHONY: deploy
deploy:
		gcloud builds submit --tag gcr.io/ory-web/slackinviter
		gcloud run deploy slackinviter --image gcr.io/ory-web/slackinviter --platform managed --region us-east1 --allow-unauthenticated
