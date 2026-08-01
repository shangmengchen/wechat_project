# ClaudeTeam 使用说明

本文档记录当前服务器上 ClaudeTeam 的启用方式、飞书使用方式、添加机器人/员工方式，以及切换工作目录的方法。

## 当前部署信息

- ClaudeTeam 源码目录：`/home/ubuntu/project/ai/ClaudeTeam`
- 当前工作目录：`/home/ubuntu/project/wechat_project`
- 当前配置文件：`/home/ubuntu/project/wechat_project/claudeteam.toml`
- 当前状态目录：`/home/ubuntu/project/wechat_project/state`
- 当前 tmux session：`ClaudeTeam-ClaudeTeam`
- 当前 agent CLI：`codex-cli`
- 当前模型：`gpt-5.5`
- 飞书模式：quick 模式，需要在群里 `@机器人`

## 1. 启用 ClaudeTeam

SSH 登录服务器后执行：

```bash
cd /home/ubuntu/project/wechat_project
export PATH=/home/ubuntu/project/ai/ClaudeTeam/.venv/bin:$PATH

claudeteam up
claudeteam health
claudeteam team
```

命令说明：

- `claudeteam up`：启动 tmux 团队、飞书 router、watchdog。
- `claudeteam health`：检查配置、tmux、router、watchdog、agent CLI。
- `claudeteam team`：查看 manager / worker 当前状态。

## 2. 停止 ClaudeTeam

```bash
cd /home/ubuntu/project/wechat_project
export PATH=/home/ubuntu/project/ai/ClaudeTeam/.venv/bin:$PATH

claudeteam down
```

## 3. 查看 tmux

查看 session：

```bash
tmux ls
```

进入 ClaudeTeam session：

```bash
tmux attach -t ClaudeTeam-ClaudeTeam
```

退出 tmux 但不关闭服务：

```text
Ctrl+B
D
```

查看 manager pane 内容：

```bash
tmux capture-pane -t ClaudeTeam-ClaudeTeam:manager -p -S -120
```

查看 worker pane 内容：

```bash
tmux capture-pane -t ClaudeTeam-ClaudeTeam:worker_codex -p -S -120
```

## 4. 飞书里怎么用

当前是 quick 模式，所以群里需要 `@机器人` 才会响应。

示例：

```text
@机器人 你好
@机器人 帮我看一下后端日志
@机器人 让团队检查今天的改动有没有问题
@机器人 帮我安排 worker 检查小程序前端 bug
```

常用飞书命令：

```text
@机器人 /health
@机器人 /team
@机器人 /tmux
@机器人 /task
@机器人 /stop
@机器人 /restart
```

说明：

- quick 模式优点：服务器无桌面环境也能通过扫码创建机器人和群。
- quick 模式限制：群里必须 `@机器人`，不支持群消息无 @ 自动响应。
- 如果以后想“不 @ 也响应”，需要改用企业自建应用模式，通常要在有桌面浏览器的机器上完成飞书控制台授权。

## 5. 添加机器人/员工

编辑配置文件：

```bash
cd /home/ubuntu/project/wechat_project
vim claudeteam.toml
```

在现有 `[team.agents.*]` 后面新增一个 agent。

示例：添加测试员工：

```toml
[team.agents.worker_test]
cli = "codex-cli"
model = "gpt-5.5"
role = "Test worker"
specialty = ["testing", "bug reproduction", "quality check"]
notes = "Always report in Chinese."
card_color = "green"
```

保存后重启团队：

```bash
cd /home/ubuntu/project/wechat_project
export PATH=/home/ubuntu/project/ai/ClaudeTeam/.venv/bin:$PATH

claudeteam down
claudeteam up
claudeteam team
```

删除机器人/员工：

1. 从 `claudeteam.toml` 删除对应 `[team.agents.xxx]` 配置块。
2. 执行 `claudeteam down && claudeteam up`。
3. 执行 `claudeteam team` 确认名单。

临时雇佣/解雇也可以使用：

```bash
claudeteam hire worker_name
claudeteam fire worker_name
```

长期固定团队建议直接修改 `claudeteam.toml`，更清晰稳定。

## 6. 切换工作目录

ClaudeTeam 的工作目录就是执行 `claudeteam up` 时所在的目录，并且该目录里需要有：

```text
claudeteam.toml
state/
```

当前工作目录是：

```bash
/home/ubuntu/project/wechat_project
```

如果要切到另一个项目，例如：

```bash
/home/ubuntu/project/other_project
```

执行：

```bash
cd /home/ubuntu/project/wechat_project
export PATH=/home/ubuntu/project/ai/ClaudeTeam/.venv/bin:$PATH
claudeteam down

cd /home/ubuntu/project/other_project
cp /home/ubuntu/project/wechat_project/claudeteam.toml .
cp -a /home/ubuntu/project/wechat_project/state .
claudeteam install-hooks
claudeteam up
claudeteam health
```

确认 pane 工作目录：

```bash
tmux capture-pane -t ClaudeTeam-ClaudeTeam:manager -p | grep 'project'
```

如果输出里看到目标目录，例如：

```text
gpt-5.5 high · ~/project/other_project
```

说明切换成功。

## 7. Codex 中转站配置

服务器上 Codex 已配置为使用中转站，配置位置包括：

```text
/home/ubuntu/.codex/config.toml
/home/ubuntu/project/wechat_project/state/agents/manager/home/.codex/config.toml
/home/ubuntu/project/wechat_project/state/agents/worker_codex/home/.codex/config.toml
```

检查登录状态：

```bash
CODEX_HOME=/home/ubuntu/.codex codex login status
CODEX_HOME=/home/ubuntu/project/wechat_project/state/agents/manager/home/.codex codex login status
CODEX_HOME=/home/ubuntu/project/wechat_project/state/agents/worker_codex/home/.codex codex login status
```

如果 Codex 不可用，优先检查：

```bash
CODEX_HOME=/home/ubuntu/.codex codex doctor
```

## 8. 推荐日常命令

```bash
cd /home/ubuntu/project/wechat_project
export PATH=/home/ubuntu/project/ai/ClaudeTeam/.venv/bin:$PATH

claudeteam health
claudeteam team
claudeteam up
claudeteam down
claudeteam restart manager
claudeteam restart worker_codex
```

## 9. 常见问题

### 飞书群里没响应

当前是 quick 模式，必须 `@机器人`。

```text
@机器人 你好
```

然后在服务器检查：

```bash
cd /home/ubuntu/project/wechat_project
export PATH=/home/ubuntu/project/ai/ClaudeTeam/.venv/bin:$PATH

claudeteam health
tail -100 state/router.log
```

### agent 不 ready

查看状态：

```bash
claudeteam team
claudeteam health
```

查看 pane：

```bash
tmux capture-pane -t ClaudeTeam-ClaudeTeam:manager -p -S -120
tmux capture-pane -t ClaudeTeam-ClaudeTeam:worker_codex -p -S -120
```

重启单个 agent：

```bash
claudeteam restart manager
claudeteam restart worker_codex
```

### 工作目录不对

先停止，再进入正确目录启动：

```bash
cd /home/ubuntu/project/wechat_project
export PATH=/home/ubuntu/project/ai/ClaudeTeam/.venv/bin:$PATH

claudeteam down
claudeteam up
```

确认：

```bash
tmux capture-pane -t ClaudeTeam-ClaudeTeam:manager -p | grep 'project'
```

