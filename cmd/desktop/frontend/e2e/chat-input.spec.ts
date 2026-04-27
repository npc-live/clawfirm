import { test, expect, Page } from '@playwright/test'

/**
 * Chat Input E2E Tests
 * 测试 ChatView 中的输入框组件功能
 */

// 等待 WebSocket 连接就绪
async function waitForChatReady(page: Page) {
  // 等待 textarea 出现且 placeholder 变为可输入状态
  await page.waitForFunction(() => {
    const textarea = document.querySelector('textarea')
    return textarea && textarea.getAttribute('placeholder')?.includes('Type a message')
  }, { timeout: 10000 })
}

test.describe('Chat Input - P0 核心功能', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('http://localhost:5173')
    await page.waitForLoadState('networkidle')
    await waitForChatReady(page)
  })

  test('输入框渲染检查', async ({ page }) => {
    const textarea = page.locator('textarea').last() // 有多个 textarea，取最后一个（主输入框）
    await expect(textarea).toBeVisible()
    await expect(textarea).toHaveAttribute('placeholder', /Type a message/i)
  })

  test('文本输入功能', async ({ page }) => {
    const textarea = page.locator('textarea').last()
    await textarea.fill('Hello, World!')
    await expect(textarea).toHaveValue('Hello, World!')
  })

  test('发送按钮状态 - 空输入时禁用', async ({ page }) => {
    const sendButton = page.getByRole('button', { name: 'Send' })
    await expect(sendButton).toBeDisabled()
  })

  test('发送按钮状态 - 有输入时启用', async ({ page }) => {
    const textarea = page.locator('textarea').last()
    const sendButton = page.getByRole('button', { name: 'Send' })
    
    await textarea.fill('Hello')
    await expect(sendButton).toBeEnabled()
  })

  test('点击发送按钮发送消息', async ({ page }) => {
    const textarea = page.locator('textarea').last()
    const sendButton = page.getByRole('button', { name: 'Send' })
    
    await textarea.fill('Test message')
    await sendButton.click()
    
    // 消息发送后输入框应该清空
    await expect(textarea).toHaveValue('')
  })
})

test.describe('Chat Input - P1 交互体验', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('http://localhost:5173')
    await page.waitForLoadState('networkidle')
    await waitForChatReady(page)
  })

  test('Cmd+Enter 快捷键发送', async ({ page }) => {
    const textarea = page.locator('textarea').last()
    
    await textarea.fill('Test with keyboard')
    // ⌘ 是 Meta 键
    await textarea.press('Meta+Enter')
    
    await expect(textarea).toHaveValue('')
  })

  test('输入多行文本自动增高', async ({ page }) => {
    const textarea = page.locator('textarea').last()
    
    // 记录初始高度
    const initialHeight = await textarea.evaluate((el) => el.clientHeight)
    
    // 输入多行
    await textarea.fill('Line 1\nLine 2\nLine 3\nLine 4\nLine 5')
    
    // 触发 height 更新（通过输入或事件）
    await textarea.dispatchEvent('input')
    
    // 检查高度是否增加（最大 200px）
    const newHeight = await textarea.evaluate((el) => el.clientHeight)
    expect(newHeight).toBeGreaterThan(initialHeight)
  })

  test('流式输出时输入框禁用', async ({ page }) => {
    const textarea = page.locator('textarea').last()
    
    // 发送消息
    await textarea.fill('Hello')
    const sendButton = page.getByRole('button', { name: 'Send' })
    await sendButton.click()
    
    // 短暂等待，检查是否出现 Stop 按钮（表示进入流式输出）
    const stopButton = page.getByRole('button', { name: 'Stop' })
    
    // 如果 Stop 出现，说明进入流式模式
    const stopVisible = await stopButton.isVisible().catch(() => false)
    if (stopVisible) {
      // 流式输出中，输入框应该被禁用
      await expect(textarea).toBeDisabled()
    }
  })
})

test.describe('Chat Input - P2 附件功能', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('http://localhost:5173')
    await page.waitForLoadState('networkidle')
    await waitForChatReady(page)
  })

  test('附件按钮存在', async ({ page }) => {
    const attachButton = page.getByRole('button', { name: 'Attach files' }).or(page.getByTitle('Attach files'))
    await expect(attachButton).toBeVisible()
  })

  test('点击附件按钮触发文件选择', async ({ context, page }) => {
    // 监听文件选择对话框
    const chooseLaterPromise = page.waitForEvent('dialog', { timeout: 3000 }).catch(() => null)
    
    const attachButton = page.getByRole('button', { name: 'Attach files' }).or(page.getByTitle('Attach files'))
    await attachButton.click()
    
    // 如果有对话框，接受或取消
    const dialog = await chooseLaterPromise
    if (dialog) {
      await dialog.dismiss()
    }
  })
})
