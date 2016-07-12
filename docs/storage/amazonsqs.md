# Exporting cAdvisor Stats to Amazon SQS

cAdvisor supports exporting stats to [Amazon SQS](https://aws.amazon.com/sqs/).

To export stats to SQS, you need to provide the additional flags to cAdvisor.

Set the storage driver as SQS:

```
 -storage_driver="amazonsqs"
```

Specify SQS Queue address:

```
 -storage_driver_amazonsqs_queue="https://sqs.eu-central-1.amazonaws.com/1234567890/container-stats-queue"
```

Specify region:

```
 -storage_driver_amazonsqs_region="eu-central-1"
```

And set environment variables:

```
AWS_ACCESS_KEY_ID=your_access_key_id
AWS_SECRET_ACCESS_KEY=your_secret_access_key
```
