后端：

语言：Go

Web 框架：Gin

ORM：GORM

数据库：MySQL，数据库名 echo_space

缓存：Redis，支持本地缓存兜底

消息队列：RabbitMQ

搜索：Elasticsearc

视频处理：FFmpeg / FFprobe

配置方式：configs/application.yaml

服务形态：Go 单体服务

—— —— —— —— —— —— —— —— —— —— —— —— ——

前端：

框架：Vue 3

构建工具：Vite

UI 组件库：Element Plus

状态管理：Pinia

路由：Vue Router

请求库：Axios

样式：SCSS / Sass

播放器：ArtPlayer

HLS 播放：hls.js

弹幕：artplayer-plugin-danmuku

图表：ECharts

其他：mitt、js-md5、vue-cropper、vue-draggable-plus、vue-cookies

—— —— —— —— —— —— —— —— —— —— —— —— ——

项目结构
echo-space

├── echo-space-backend          Go 后端单体服务

├── echo-space-frontend-admin   管理端前端

├── echo-space-frontend-web     用户端前端

├── README.md                   项目展示说明

└── AGENTS.md                   AI 开发规范

—— —— —— —— —— —— —— —— —— —— —— —— ——

端口说明
用户端前端：http://localhost:3000
管理端前端：http://localhost:3001
Go 后端服务：http://localhost:7070

—— —— —— —— —— —— —— —— —— —— —— —— ——

访客端：
登录页面
<img width="1912" height="981" alt="image" src="https://github.com/user-attachments/assets/31736197-21fa-4df3-88a1-ddefffd77499" />
注册页面
<img width="1912" height="979" alt="image" src="https://github.com/user-attachments/assets/6eba80d4-801f-4f6e-83a8-0f97a8317d4e" />
首页：
<img width="1918" height="978" alt="image" src="https://github.com/user-attachments/assets/8b1021ec-d898-47f1-adc9-fffaf99e18f7" />
图库：
<img width="1919" height="986" alt="image" src="https://github.com/user-attachments/assets/98591218-b560-45ac-94a4-6783fab21d7a" />
周边列表页面
<img width="1905" height="982" alt="image" src="https://github.com/user-attachments/assets/0044a642-cd3d-412d-a8ae-1b18b9dd78b5" />
周边详情页面
<img width="1889" height="981" alt="image" src="https://github.com/user-attachments/assets/4fa97438-4684-441c-937d-6beb31064bd3" />
<img width="1908" height="981" alt="image" src="https://github.com/user-attachments/assets/50379aca-220a-40dc-8017-aca69545e982" />
下单后支付页面
<img width="1908" height="983" alt="image" src="https://github.com/user-attachments/assets/2bab2625-e942-43b7-95fb-866efe7c4c7b" />
订单列表页面
<img width="1919" height="985" alt="image" src="https://github.com/user-attachments/assets/c45330e6-77b1-4397-b498-de3cbe9847bc" />
<img width="1919" height="980" alt="image" src="https://github.com/user-attachments/assets/dd1a87a4-e789-444a-84e9-c5ff1a219b50" />
上传视频页面
<img width="1919" height="974" alt="image" src="https://github.com/user-attachments/assets/73e244da-1c1b-4a8b-844f-98ef957da413" />
<img width="1919" height="980" alt="image" src="https://github.com/user-attachments/assets/4fea49f5-427f-4851-88d4-23f67ae82260" />
上传图片页面
<img width="1919" height="986" alt="image" src="https://github.com/user-attachments/assets/c6c5de6d-a208-4eff-b939-289ecf74b7d2" />
稿件管理页面
<img width="1917" height="987" alt="image" src="https://github.com/user-attachments/assets/b7e196ad-fe13-42ba-a343-071175275e85" />
<img width="1919" height="991" alt="image" src="https://github.com/user-attachments/assets/6bb5188a-2060-4be8-9dd8-4e469317c0b0" />
观看视频以及互动页面
<img width="1919" height="983" alt="image" src="https://github.com/user-attachments/assets/ea51ff24-01be-49ed-bb3b-837500888889" />
评论页面
<img width="1919" height="984" alt="image" src="https://github.com/user-attachments/assets/0df5952b-5a09-4ed8-a87f-6b66ea8b968c" />
<img width="1914" height="983" alt="image" src="https://github.com/user-attachments/assets/4fa63ab3-13cd-4daa-8f1a-656d7f1c445a" />
动态页面
<img width="1919" height="976" alt="image" src="https://github.com/user-attachments/assets/1dd6b07a-b79b-4e69-bff9-919fbe6975da" />
图片详情：
<img width="1919" height="980" alt="image" src="https://github.com/user-attachments/assets/cb0d7624-c76f-4b21-85db-3169e8c6952c" />
搜索视频页面
<img width="1919" height="982" alt="image" src="https://github.com/user-attachments/assets/f6388f5d-1581-4791-aafd-6999e0eb8c18" />
个人主页周边展示柜
<img width="1919" height="985" alt="image" src="https://github.com/user-attachments/assets/ea3854f2-44e1-4a04-af65-e4a103d7462c" />
<img width="1919" height="989" alt="image" src="https://github.com/user-attachments/assets/06701344-6d84-489b-99f5-002d081487c2" />
创作中心数据统计页面
<img width="1919" height="981" alt="image" src="https://github.com/user-attachments/assets/ae2ce8aa-1674-4a28-a4e7-68a5e544040c" />
创作中心评论/弹幕管理页面
<img width="1919" height="979" alt="image" src="https://github.com/user-attachments/assets/d8a1a8d0-227a-4f33-bbbf-3f871c40e08a" />
<img width="1919" height="987" alt="image" src="https://github.com/user-attachments/assets/e802467b-92d2-4a92-857b-8def1d67012e" />
<img width="1919" height="980" alt="image" src="https://github.com/user-attachments/assets/e4ff6bbc-5538-4469-b4c9-55b299775a1e" />

—— —— —— —— —— —— —— —— —— —— —— —— ——

管理端：
登录页面
<img width="1919" height="981" alt="image" src="https://github.com/user-attachments/assets/0f52b1b6-b4b9-48eb-bded-7193a6b64002" />
用户管理页面
<img width="1917" height="974" alt="image" src="https://github.com/user-attachments/assets/1839609b-7865-462f-8a1d-ba1d469fc99c" />
周边管理页面
<img width="1919" height="976" alt="image" src="https://github.com/user-attachments/assets/aff093fe-2ccc-4614-959e-4fa231e9ea4f" />
<img width="1919" height="977" alt="image" src="https://github.com/user-attachments/assets/3f30dc6e-8653-4103-9d40-361739304721" />
新增周边页面
<img width="1909" height="977" alt="image" src="https://github.com/user-attachments/assets/d6ed6a3b-07a4-49cf-9ecb-b29dfacb4174" />
稿件管理页面
<img width="1919" height="979" alt="image" src="https://github.com/user-attachments/assets/fcaa26bc-4ca4-4164-ae2e-3de07d0d6bee" />
<img width="1919" height="986" alt="image" src="https://github.com/user-attachments/assets/7bad663e-36a5-4bae-8a13-838cc1f7d959" />
<img width="1919" height="983" alt="image" src="https://github.com/user-attachments/assets/a06a2b06-5a86-4524-8b10-db772a963871" />
<img width="1919" height="980" alt="image" src="https://github.com/user-attachments/assets/f8c81ba9-70c2-4e91-8dd7-4799b0fe89fb" />
审核详情页面
<img width="1919" height="984" alt="image" src="https://github.com/user-attachments/assets/ab415266-19ea-4d7b-9453-ccce7ca9968f" />
<img width="1919" height="987" alt="image" src="https://github.com/user-attachments/assets/7fe67c3d-80a9-4718-b9d3-9c17d252f423" />

