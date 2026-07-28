import { expect, test, type Page, type TestInfo } from '@playwright/test'

function observeBrowser(page: Page, testInfo: TestInfo) {
  const consoleErrors: string[] = []
  const failedRequests: string[] = []

  page.on('console', message => {
    if (message.type() === 'error') consoleErrors.push(message.text())
  })
  page.on('pageerror', error => consoleErrors.push(`pageerror: ${error.message}`))
  page.on('requestfailed', request => {
    const url = request.url()
    if (url.startsWith('http://127.0.0.1:') || url.startsWith('http://localhost:')) {
      failedRequests.push(`${request.method()} ${url}: ${request.failure()?.errorText ?? 'unknown error'}`)
    }
  })
  page.on('response', response => {
    const url = response.url()
    if (response.status() >= 400 && (url.startsWith('http://127.0.0.1:') || url.startsWith('http://localhost:'))) {
      failedRequests.push(`${response.request().method()} ${url}: HTTP ${response.status()}`)
    }
  })

  return async () => {
    await testInfo.attach('browser-diagnostics', {
      body: JSON.stringify({ consoleErrors, failedRequests }, null, 2),
      contentType: 'application/json',
    })
    expect(consoleErrors, 'unexpected browser console errors').toEqual([])
    expect(failedRequests, 'unexpected failed application requests').toEqual([])
  }
}

test('loads a daily session and opens its lesson', async ({ page }, testInfo) => {
  const assertBrowserClean = observeBrowser(page, testInfo)

  try {
    await page.goto('/today')
    await expect(page.getByRole('heading', { name: /Good evening, .+\./ })).toBeVisible()
    await expect(page.getByText("Today’s focus")).toBeVisible()
    await page.getByRole('link', { name: /Continue session/ }).click()
    await expect(page).toHaveURL(/\/lesson$/)
    await expect(page.getByRole('link', { name: /Start knowledge check/ })).toBeVisible()
  } finally {
    await assertBrowserClean()
  }
})
