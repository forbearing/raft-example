### 启动



#### 第一个节点(leader)启动

```bash
go run . --id node0 ./node0
```



#### 其他节点启动

```bash
# 节点1
go run . --id node1 --haddr localhost:13001 --raddr localhost:14001 --join :13000 ./node1
go run . --id node2 --haddr localhost:13002 --raddr localhost:14002 --join :13000 ./node2
go run . --id node3 --haddr localhost:13003 --raddr localhost:14003 --join :13000 ./node3
go run . --id node4 --haddr localhost:13004 --raddr localhost:14004 --join :13000 ./node4
```



### 接口

#### Set Store

```bash
curl -XPOST http://localhost:13000/store -d '{"key1":"value1"}'
```

#### Get Store

```bash
curl -XGET http://localhost:13000/store/key1
```

#### Status

```bash
curl -XGET http://localhost:13000/status | jq
```

```json
{
  "me": {
    "id": "",
    "address": "localhost:14000"
  },
  "leader": {
    "id": "node1",
    "address": "127.0.0.1:14001"
  },
  "followers": [
    {
      "id": "node0",
      "address": "127.0.0.1:14000"
    },
    {
      "id": "node2",
      "address": "localhost:14002"
    },
    {
      "id": "node3",
      "address": "localhost:14003"
    },
    {
      "id": "node4",
      "address": "localhost:14004"
    }
  ]
}
```





