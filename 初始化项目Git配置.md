# 初始化项目Git配置

请按照以下步骤初始化Git仓库并上传到我的Gitea服务器：

## 步骤

1. **初始化Git仓库**
   ```bash
   git init
   ```

2. **创建main分支**
   ```bash
   git checkout -b main
   ```

3. **创建.gitignore文件**，忽略不必要的内容：
   - Node/React: node_modules/, build/, coverage/
   - Go: *.exe, *.test, *.out, *.dll, *.so, *.dylib
   - 数据文件: data/data.json
   - 日志: *.log
   - 操作系统: .DS_Store, Thumbs.db

4. **添加所有代码文件并提交**
   ```bash
   git add .
   git commit -m "first commit"
   ```

5. **添加Gitea远程仓库**
   ```bash
   git remote add gitea ssh://git@repo.shumengya.top:8022/{{USER}}/{{REPO}}.git
   ```

6. **推送到Gitea**
   ```bash
   git push -u gitea main
   ```

## 说明

- Gitea服务器地址：`repo.shumengya.top:8022`
- 使用SSH协议上传
- 仓库路径：修改 `{{USER}}/{{REPO}}` 为你的用户名和仓库名
