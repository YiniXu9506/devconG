# PingCAP Account API 文档

- [Fetch Phrases](#fetch-phrases)
- [Add a New Phrase](#add-new-phrase)
- [Submit Phrase Click Info](#submit-phrase)

### Fetch Phrases

#### Request

- Method: **GET**
- URL: `/p`

#### Response Example

- Success

  - HTTP/1.1 200

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

- Failed

  - HTTP/1.1 401

    ```json
    {
      "detail": "Authentication credentials were not provided."
    }
    ```
