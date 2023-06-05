# 腾讯云函数 chatGPT 反向代理服务

使用腾讯云函数来搭建 chatGPT 反向代理服务，请求 api 时，直接把接口地址 ( https://api.openai.com ) 替换为腾讯云函数的地址。

### A. 新建云函数

1. 进入腾讯云函数控制台: https://console.cloud.tencent.com/scf/list?rid=15&ns=default
2. “函数服务” --> “新建”
   - 点击 “从头开始”
   - 基础配置
      - 函数类型: Web函数
      - 名称: 随便填；例如：chatGPT
      - 地域: 选择境外的美国、加拿大等，推荐“硅谷”
      - 运行环境: Go 1
      - 时区: Asia/Shanghai(北京时间)
   - 函数代码
      - 提交方法: 本地上传zip包

        zip 包文件名: tencentcloud_chatgpt_reverse_proxy_v0.x.x_linux_amd64.zip

        下载地址: https://github.com/lenye/chatgpt_reverse_proxy/releases
   - 高级配置
      - 启动命令: 自定义模板
   - 环境配置
      - 内存: 128MB
      - 执行超时时间: 180 秒
   - 点击 “完成”

![基础配置.png](docs/new.png)

![高级配置.png](docs/new2.png)

### B. 函数管理

1. 进入腾讯云函数控制台: https://console.cloud.tencent.com/scf/list?rid=15&ns=default
2. “函数服务” --> 在函数列表中选择刚刚新建函数“chatGPT”
3. “函数管理” --> “函数代码”
    - 访问路径

      复制链接: https://service-xxx-xxx.xxx.apigw.tencentcs.com/release/

![访问路径.png](docs/new3.png)

### C. chatGPT 反向代理服务，腾讯云函数的地址

访问路径去除 "/release/"，得到 chatGPT 反向代理服务，腾讯云函数的地址:

https://service-xxx-xxx.xxx.apigw.tencentcs.com

请求 chatGPT api 时，直接把接口地址 ( https://api.openai.com ) 替换为腾讯云函数的地址。

### D. 使用样例

#### go

```go
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
)

func main() {
	cfg := openai.DefaultConfig(os.Getenv("OPENAI_API_KEY"))

	// 修改BaseURL为腾讯云函数的地址，不要忘记"/v1"
	cfg.BaseURL = "https://service-xxx-xxx.xxx.apigw.tencentcs.com/v1"

	client := openai.NewClientWithConfig(cfg)

	req := openai.ChatCompletionRequest{
		Model: openai.GPT3Dot5Turbo,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: "you are a helpful chatbot",
			},
		},
	}
	fmt.Println("Conversation")
	fmt.Println("---------------------")
	fmt.Print("> ")
	s := bufio.NewScanner(os.Stdin)
	for s.Scan() {
		req.Messages = append(req.Messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: s.Text(),
		})
		resp, err := client.CreateChatCompletion(context.Background(), req)
		if err != nil {
			fmt.Printf("ChatCompletion error: %v\n", err)
			continue
		}
		fmt.Printf("%s\n\n", resp.Choices[0].Message.Content)
		req.Messages = append(req.Messages, resp.Choices[0].Message)
		fmt.Print("> ")
	}
}

```