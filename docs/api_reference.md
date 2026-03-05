# "在吗" APP - REST API 接口文档

**Base URL**: `http://localhost:8080/api/v1`

**通用返回格式 (JSON)**:
```json
{
  "code": 0,          // 0 为成功，非 0 为业务错误码
  "message": "success", // 错误时的详细提示
  "data": { ... }      // 业务数据，可能为对象或数组
}
```

---

## 1. 认证模块 (Auth)

### 1.1 发送短信验证码
- **URL**: `/auth/sms-code`
- **Method**: `POST`
- **鉴权**: 免鉴权
- **Body**:
  ```json
  {
    "phone": "13800138000" // 必须为11位
  }
  ```
- **Response**: 返回成功提示，开发环境下会直接在 `data` 中返回验证码。

### 1.2 登录/注册
- **URL**: `/auth/login`
- **Method**: `POST`
- **鉴权**: 免鉴权
- **Body**:
  ```json
  {
    "phone": "13800138000",
    "code": "123456",
    "role": 1,           // 1=老人, 2=年轻人
    "device_id": "xxx"   // 可选，设备标识，用于老人端异常设备登录告警
  }
  ```
- **Response**: 返回成功会提取 `token`。格式如下：
  ```json
  {
    "code": 0,
    "message": "success",
    "data": {
      "is_new": true,
      "role": 1,
      "token": "eyJhbGciOiJIUzI1NiIs...",
      "user_id": 12,
      "nickname": "用户12"
    }
  }
  ```
  后续所有需鉴权的接口请在 HTTP Header 中加入：`Authorization: Bearer <token>`。

---

## 2. 用户资料与关系模块 (User)

> 以下接口均需要在 Header 中携带 `Authorization: Bearer <token>`

### 2.1 获取当前用户信息
- **URL**: `/user/profile`
- **Method**: `GET`
- **Response**: 返回当前用户信息、绑定的亲属关系列表以及兴趣标签。

### 2.2 更新用户资料
- **URL**: `/user/profile`
- **Method**: `PUT`
- **Body**:
  ```json
  {
    "nickname": "张大爷", // 必须<=8字
    "avatar_url": "https://zaima.oss.aliyuncs.com/avatar.jpg", // ⚠️ 必须为合法的 HTTPS OSS 链接
    "city": "武汉",
    "province": "湖北"
  }
  ```

### 2.3 发起亲子绑定
- **URL**: `/user/bind`
- **Method**: `POST`
- **Body**:
  ```json
  {
    "target_phone": "13912345678",
    "youth_city": "深圳", // (可选) 年轻人填
    "elder_addr": "中山路", // (可选) 长辈填
    "remark": "儿子"
  }
  ```
- **Response**: 状态可能是 `pending` (等待确认) 或者是 `not_registered` (对方未注册)。

### 2.4 确认绑定
- **URL**: `/user/bind/confirm`
- **Method**: `POST`
- **Body**:
  ```json
  {
    "relation_id": 123
  }
  ```

### 2.5 更新兴趣标签 (长辈端)
- **URL**: `/user/interests`
- **Method**: `PUT`
- **Body**:
  ```json
  {
    "tags": ["广场舞", "下棋", "散步"] // 最多3个
  }
  ```

---

## 3. 设备监控与 AI 报告 (Device)

### 3.1 每日数据上报 (长辈端静默)
- **URL**: `/device/upload`
- **Method**: `POST`
- **Body**:
  ```json
  {
    "record_date": "2026-03-05",
    "steps": 5300,
    "battery_level": 45,
    "screen_unlocks": 15,
    "screen_usage_mins": 120
  }
  ```

### 3.2 获取长辈今日状态与 AI 建议 (年轻人端)
- **URL**: `/device/daily-insight?elder_id=123`
- **Method**: `GET`
- **Response**: 返回长辈的当日数据、历史数据 (最多7天) 和基于大模型生成的关切提示语。

### 3.3 获取长辈月度 AI 报告
- **URL**: `/device/monthly-report?elder_id=123&month=2026-03`
- **Method**: `GET`
- **Response**: 返回 AI 生成的生活轨迹总结与健康提醒。

---

## 4. “一起玩” 广场交友 (Square)

### 4.1 发布兴趣气泡
- **URL**: `/square/publish`
- **Method**: `POST`
- **Body**:
  ```json
  {
    "voice_url": "https://zaima.oss.aliyuncs.com/voice.mp3", // ⚠️ 必须为合法的 HTTPS OSS 链接
    "interest_tag": "打门球",
    "province": "湖北省",
    "city": "武汉市",
    "longitude": 114.3055,
    "latitude": 30.5928
  }
  ```
- **说明**: 气泡存活时间为 4 小时。

### 4.2 发现广场气泡 (带筛选与搜索)
- **URL**: `/square/bubbles?lng=xxx&lat=xxx&radius=10000&province=xxx&city=xxx&keyword=xxx`
- **Method**: `GET`
- **说明**: 结合 GEO 位置查询周围的气泡，同时支持省市下拉过滤和关键字 (昵称) 搜索。

### 4.3 确认匹配意向
- **URL**: `/square/match-confirm`
- **Method**: `POST`
- **Body**:
  ```json
  {
    "bubble_id": 456
  }
  ```
- **说明**: 成功匹配后气泡消失，服务端通过 WebSocket 通知其他在等待的请求者。

---

## 5. 辅助与工具接口

### 5.1 获取天气信息
- **URL**: `/weather?city=武汉`
- **Method**: `GET`

### 5.2 获取 AI 关怀卡片
- **URL**: `/weather/care-cards?city=武汉`
- **Method**: `GET`
- **说明**: 返回贴合当前时间段和天气的短句及 Icon，用于聊天列表置顶展示。

### 5.3 获取新闻列表
- **URL**: `/news?city=武汉&tab=latest&keyword=搜索词&page=1&page_size=10`
- **Method**: `GET`
- **说明**: 
  - 支持热点 (hot)/最新 (latest)/全部 (all) 切换。
  - 支持关键字 (`keyword`) 模糊匹配搜索。
  - 支持页码 (`page`) 和每页数量 (`page_size`) 分页。

---

## 6. 聊天模块 (Chat REST)

### 6.1 获取会话列表 (首屏)
- **URL**: `/chat/sessions?keyword=xxx`
- **Method**: `GET`
- **Response**: 返回所有最近聊天的对象、系统消息、未读数量及最后一条消息内容的摘要。

### 6.2 获取聊天记录
- **URL**: `/chat/history?peer_id=456&since_id=100021&page_size=50`
- **Method**: `GET`
- **说明**: 
  - 通过 `since_id` 传递上次拉取的最后一条 `MessageID` 以获取断网期间丢失的消息。
  - 获取后会自动将对方离线发送的消息标为已读。

### 6.3 AI 一键回复建议 (年轻人端)
- **URL**: `/chat/ai-suggest?peer_id=456`
- **Method**: `POST`
- **说明**: 结合最近的聊天上下文（通过大模型或规则引擎）分析，返回建议的破冰回复。

### 6.4 发送端语音转文字 (长辈端 STT)
- **URL**: `/chat/stt`
- **Method**: `POST`
- **Body Header**: `Content-Type: application/x-www-form-urlencoded`
  - `voice_url=https://zaima.oss.aliyuncs.com/voice.mp3` // ⚠️ 必须为合法的 HTTPS OSS 链接
- **说明**: 用于长辈滑动手势录音上传到 OSS 后，提交给服务端识别为文字后发送。
