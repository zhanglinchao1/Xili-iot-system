# Edge设备管理系统 - Web前端

## 简介

这是Edge储能柜边缘认证网关的Web管理界面，提供设备管理、监控和配置功能。

## 功能特性

### ✅ 已实现功能

1. **设备监控仪表板**
   - 设备总览统计
   - 在线/离线/故障设备统计
   - 传感器类型分布展示
   - 实时状态更新

2. **设备管理**
   - 设备列表查看（支持分页）
   - 设备注册
   - 设备信息编辑
   - 设备删除
   - 设备详情查看
   - 状态筛选（在线/离线/禁用/故障）
   - 传感器类型筛选
   - 设备搜索

3. **用户界面**
   - 现代化响应式设计
   - 深色侧边栏导航
   - 实时Toast通知
   - 模态框表单
   - 加载状态提示
   - 移动端适配

### 🚧 待开发功能

- 统计分析页面
- 系统设置页面
- 数据采集监控
- 告警管理
- 用户认证登录

## 技术栈

- **HTML5** - 语义化标签
- **CSS3** - 现代化样式、Grid布局、Flexbox
- **原生JavaScript (ES6+)** - 无框架依赖
- **Font Awesome 6** - 图标库

## 项目结构

```
web/
├── index.html          # 主HTML文件
├── css/
│   └── styles.css      # 样式文件
├── js/
│   ├── api.js          # API调用封装
│   ├── ui.js           # UI工具函数
│   ├── devices.js      # 设备管理逻辑
│   └── app.js          # 主应用入口
├── assets/             # 静态资源（图片等）
└── README.md           # 本文档
```

## 快速开始

### 1. 启动后端服务

确保Edge后端服务已启动：

```bash
cd /home/uestc/Edge
./bin/edge -config ./configs/config.yaml
```

### 2. 访问前端

由于是纯静态前端，可以通过以下方式访问：

#### 方法A: 使用Python简易服务器（推荐）

```bash
cd /home/uestc/Edge/web
python3 -m http.server 8000
```

然后在浏览器访问：`http://localhost:8000`

#### 方法B: 使用Node.js serve

```bash
# 安装serve（如果未安装）
npm install -g serve

# 启动服务
cd /home/uestc/Edge/web
serve -p 8000
```

#### 方法C: 直接用浏览器打开

```bash
# 在浏览器中打开
file:///home/uestc/Edge/web/index.html
```

**注意**: 直接打开文件可能会遇到CORS问题，建议使用HTTP服务器。

### 3. 配置API地址

默认API地址是 `http://localhost:8001`。如果后端地址不同，请修改 `js/api.js` 中的 `baseURL`：

```javascript
const API = {
    baseURL: 'http://your-backend-address:port',
    // ...
};
```

## API接口文档

前端调用的API接口请参考：
- 设备管理接口：`../device.md`
- 完整API文档：`../docs/frontend-design/CONTRACTS-API.md`

## 开发指南

### 添加新功能

1. 在 `index.html` 中添加页面结构
2. 在 `css/styles.css` 中添加样式
3. 在对应的 `js/*.js` 文件中添加逻辑
4. 在 `app.js` 中注册页面路由

### 调试技巧

打开浏览器控制台，可以使用以下调试命令：

```javascript
// 创建测试设备
App.generateTestDevice()

// 重新加载设备列表
DeviceManager.loadDevices()

// 重新加载统计信息
DeviceManager.loadStatistics()

// 健康检查
API.healthCheck()
```

### 浏览器兼容性

支持现代浏览器：
- Chrome 90+
- Firefox 88+
- Safari 14+
- Edge 90+

## 常见问题

### Q: 无法连接到后端服务？
A: 检查：
1. 后端服务是否启动
2. API地址配置是否正确
3. 是否有CORS问题（查看浏览器控制台）
4. 确认使用的是更新后的后端版本（设备API无需认证）

### Q: 设备列表为空？
A: 可能原因：
1. 数据库中还没有设备数据
2. 使用 `App.generateTestDevice()` 创建测试设备
3. 或通过API手动注册设备

### Q: 修改代码后没有生效？
A: 尝试：
1. 清除浏览器缓存（Ctrl+Shift+R 强制刷新）
2. 检查浏览器控制台是否有错误
3. 确认修改的文件已保存

## 性能优化

- 使用防抖（debounce）优化搜索输入
- 分页加载避免一次性加载大量数据
- 懒加载和按需加载
- 定期自动刷新（可配置间隔）

## 安全考虑

- **当前版本**: 设备管理API无需认证，专为内网管理使用
- **生产环境建议**: 配置IP白名单、VPN或反向代理认证
- XSS防护（用户输入转义）
- CSRF防护
- 敏感信息不在前端存储

**注意**: 当前版本适用于内网环境，如需部署到公网，请配置适当的访问控制。

## 贡献指南

欢迎提交Issue和Pull Request！

## 许可证

与主项目保持一致

---

**维护团队**: Edge Development Team  
**最后更新**: 2025-10-15
