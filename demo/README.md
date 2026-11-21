# 企业 AI 中台 · 交互演示

在一处管理模型渠道、成员额度和调用审计。设计上，管理与审计留在企业私域，获准的模型请求可发往公网模型服务。本页为纯前端模拟，不是可用于生产的网关。

## 体验流程

1. **模型渠道**：启停通用或备用渠道，在请求沙盒里验证停用结果。
2. **成员与额度**：编辑成员的公司额度，发起请求后核对用量变化。
3. **出域策略**：切换敏感样例拦截、个人额度许可，核对请求的放行与拒绝。
4. **调用审计**：在右侧沙盒发起模拟调用，再到审计按结果筛选。

直接用浏览器打开 `index.html` 即可，无需安装依赖、构建或运行后端。演示仅使用虚构数据，不调用模型 API、不发送业务网络请求，也不需要 API 密钥。用页面上的「重置」恢复当前浏览器会话的初始示例数据。

## 发布到 GitHub Pages

1. 将 `index.html`、`README.md` 和空文件 `.nojekyll` 放在发布分支的根目录，提交并推送。
2. 打开仓库的 **Settings → Pages**，在 **Source** 选择 **Deploy from a branch**，选中发布分支和 **/(root)**，保存。
3. 部署完成后，在 **Settings → Pages** 点击 **Visit site**。

[GitHub Pages 发布源配置](https://docs.github.com/en/pages/getting-started-with-github-pages/configuring-a-publishing-source-for-your-github-pages-site) · [入口文件与静态部署说明](https://docs.github.com/en/pages/getting-started-with-github-pages/creating-a-github-pages-site)
