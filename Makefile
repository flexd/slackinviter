.PHONY: deploy
deploy:
		gcloud builds submit --project ory-web --tag gcr.io/ory-web/slackinviter
		gcloud run deploy slackinviter --project ory-web --image gcr.io/ory-web/slackinviter --platform managed --region us-east1 --allow-unauthenticated
