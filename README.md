# mm-scoring

## Install, build

```bash
# install
go install

# or build
go build
```

## How to Run
```bash

mm-scoring {start height} [flags]

# example
mm-scoring 1987252 --grpc 127.0.0.1:9090 --rpc tcp://127.0.0.1:26657 --sim true 
```


## Dashboard API

- `Score` http://127.0.0.1:8080/score

## Example output

```json
{
  "Block": {
    "Height": 1996000,
    "Time": "2022-09-17T08:20:35.134272Z"
  },
  "Score": {
    "1": {
      "cre1dmdswwz59psqxeuswyygr6x4n7mjhq7c7ztw5k": {
        "Address": "cre1dmdswwz59psqxeuswyygr6x4n7mjhq7c7ztw5k",
        "PairId": 1,
        "Eligible": false,
        "Apply": false,
        "SerialDownTime": 0,
        "TotalDownTime": 0,
        "DownThisHour": false,
        "TotalLiveHours": 19,
        "LiveHours": 19,
        "LiveDays": 0,
        "ThisMonthEligibility": false,
        "Score": "321.471975531254089096",
        "ScoreRatio": "0.225542001255679847"
      },
      "cre1fckkusk84mz4z2r4a2jj9fmap39y6q9dw3g5lk": {
        "Address": "cre1fckkusk84mz4z2r4a2jj9fmap39y6q9dw3g5lk",
        "PairId": 1,
        "Eligible": false,
        "Apply": false,
        "SerialDownTime": 0,
        "TotalDownTime": 0,
        "DownThisHour": false,
        "TotalLiveHours": 25,
        "LiveHours": 25,
        "LiveDays": 0,
        "ThisMonthEligibility": false,
        "Score": "863.842816077581710359",
        "ScoreRatio": "0.606064765634719993"
      },
      "cre1qgutsvynw88v0tjjcvjyqz6lnhzkyn8duv3uev": {
        "Address": "cre1qgutsvynw88v0tjjcvjyqz6lnhzkyn8duv3uev",
        "PairId": 1,
        "Eligible": false,
        "Apply": false,
        "SerialDownTime": 0,
        "TotalDownTime": 0,
        "DownThisHour": false,
        "TotalLiveHours": 17,
        "LiveHours": 10,
        "LiveDays": 0,
        "ThisMonthEligibility": false,
        "Score": "240.016072449720241937",
        "ScoreRatio": "0.168393233109600158"
      }
    },
    "8": {
      "cre1dmdswwz59psqxeuswyygr6x4n7mjhq7c7ztw5k": {
        "Address": "cre1dmdswwz59psqxeuswyygr6x4n7mjhq7c7ztw5k",
        "PairId": 8,
        "Eligible": false,
        "Apply": false,
        "SerialDownTime": 2059,
        "TotalDownTime": 2059,
        "DownThisHour": true,
        "TotalLiveHours": 0,
        "LiveHours": 0,
        "LiveDays": 0,
        "ThisMonthEligibility": false,
        "Score": "0.000000000000000000",
        "ScoreRatio": "0.000000000000000000"
      },
      "cre1fckkusk84mz4z2r4a2jj9fmap39y6q9dw3g5lk": {
        "Address": "cre1fckkusk84mz4z2r4a2jj9fmap39y6q9dw3g5lk",
        "PairId": 8,
        "Eligible": false,
        "Apply": false,
        "SerialDownTime": 551,
        "TotalDownTime": 551,
        "DownThisHour": true,
        "TotalLiveHours": 3,
        "LiveHours": 3,
        "LiveDays": 0,
        "ThisMonthEligibility": false,
        "Score": "1.803684095203382114",
        "ScoreRatio": "1.000000000000000000"
      },
      "cre1qgutsvynw88v0tjjcvjyqz6lnhzkyn8duv3uev": {
        "Address": "cre1qgutsvynw88v0tjjcvjyqz6lnhzkyn8duv3uev",
        "PairId": 8,
        "Eligible": false,
        "Apply": false,
        "SerialDownTime": 1940,
        "TotalDownTime": 1940,
        "DownThisHour": true,
        "TotalLiveHours": 0,
        "LiveHours": 0,
        "LiveDays": 0,
        "ThisMonthEligibility": false,
        "Score": "0.000000000000000000",
        "ScoreRatio": "0.000000000000000000"
      }
    }
  }
}
```
