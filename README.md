# PingCAP Account API 文档

- [Fetch Phrases](#fetch-phrases)
- [Add a New Phrase](#add-a-new-phrase)
- [Submit Phrase Click Info](#submit-phrase-click-info)

### Fetch Phrases

#### Request

- Method: **GET**
- URL: `/phrases`
- Query: `?limit=100`

#### Response Example

  ```json
  {
      "c": 0,
      "m": "",
      "d": [
          {
          // phrase_id int
          "phrase_id": <phrase_id>,
          // text string
          "text": '<phrase_text>',
          // hot_group_id int, 贡献最大的阵营（颜色）
          "hot_group_id": <hot_group_id>,
          // hot_group_clicks int, 贡献最大的阵营贡献的点击数
          "hot_group_clicks": <hot_group_clicks>,
          // clicks int，总点击次数（大小）
          "clicks": <click_count>,
          // update_time int, 时间戳，秒
          "update_time": < update_time_second>
          },
      ]
  }
  ```
### Add a New Phrase

#### Request

- Method: **POST**
- URL: `/phrase`,

#### Request Example

```json
{
  // 词条名
  "text": "Hello!",
  // wx open id
  "open_id": "123456789",
  // 获取到的用户 group id
  "group_id": 1
}
```

#### Response Example

- Success

  ```json
  {
    "c": 0,
    "m": "",
    "d": ""
  }
  ```

- Fail

```json

// 重名校验
{
  "c": 10001,
  "d": "",
  "m": "An existing item already exists"
}

// 不符合要求的词条名（后端校验）
{
  "c": 10002,
  "d": "",
  "m": "Maximum 10 characters"
}
```


### Submit Phrase Click Info

- Method: **POST**
- URL: `/phrase_hot`,

#### Request Example

```json
[
  {
    // phrase_id
    "phrase_id": 1,
    // wx open id
    "open_id": "123456789"，
    // group_id
    "group_id": 1,
    // 点击次数
    "clicks": 2,
  },
  ... ...
]
```

#### Response Example

- Success

  ```json
  {
    "c": 0,
    "m": "",
    "d": ""
  }
  ```
