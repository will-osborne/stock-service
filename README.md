# Usage
An HTTP GET to `/` will return the previous N days of stock close values for stock X as well as an average close value.

Weekends are omitted from the count of N days and are also not included in the response or average. Meaning if today is
Tuesday and N is 4 the response will include data for Monday, Friday, Thursday and Wednesday.

N and X are configurable in manifest/config.yaml

# Build & Publish
With docker installed and in the root of this project run:
```shell
docker build -t <repository>/stocksvc:<tag> .
docker push -t <repository>/stocksvc:<tag>
```

# Configuration & Deployment
1. Edit the manifest/config.yaml to configure the stock symbol, and the number of days to get data for.
2. Create a secret containing the api key to alphavantage
```shell
kubectl create secret generic stocksvc-api-key --from-literal=stockAPIKey=<VALUE>
```
3. Deploy the manifest
```shell
kubectl create -f manifest
```
4. Verify the deployment
```shell
kubectl get deployments -l app=stocksvc
kubectl get service -l app=stocksvc
```