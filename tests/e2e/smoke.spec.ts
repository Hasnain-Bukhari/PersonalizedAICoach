import { expect, test, type Page, type TestInfo } from '@playwright/test';

const demoMode = process.env.E2E_DEMO_MODE === 'true';

function observeBrowser(page: Page, testInfo: TestInfo, expectedFailures: RegExp[] = []) {
  const consoleErrors: string[] = [];
  const failedRequests: string[] = [];
  const recordFailure = (value: string) => {
    if (!expectedFailures.some((pattern) => pattern.test(value))) failedRequests.push(value);
  };
  page.on('console', (message) => {
    if (message.type() === 'error') consoleErrors.push(message.text());
  });
  page.on('pageerror', (error) => consoleErrors.push(`pageerror: ${error.message}`));
  page.on('requestfailed', (request) => {
    const url = request.url();
    if (url.startsWith('http://127.0.0.1:') || url.startsWith('http://localhost:'))
      recordFailure(
        `${request.method()} ${url}: ${request.failure()?.errorText ?? 'unknown error'}`
      );
  });
  page.on('response', (response) => {
    const url = response.url();
    if (
      response.status() >= 400 &&
      (url.startsWith('http://127.0.0.1:') || url.startsWith('http://localhost:'))
    )
      recordFailure(`${response.request().method()} ${url}: HTTP ${response.status()}`);
  });
  return async () => {
    await testInfo.attach('browser-diagnostics', {
      body: JSON.stringify({ consoleErrors, failedRequests }, null, 2),
      contentType: 'application/json',
    });
    expect(consoleErrors, 'unexpected browser console errors').toEqual([]);
    expect(failedRequests, 'unexpected failed application requests').toEqual([]);
  };
}

test('loads a daily session and opens its lesson', async ({ page }, testInfo) => {
  const assertBrowserClean = observeBrowser(page, testInfo);
  try {
    await page.goto('/today');
    await expect(page.getByRole('heading', { name: /Good evening, .+\./ })).toBeVisible();
    await expect(page.getByText('Today’s focus')).toBeVisible();
    await page.getByRole('link', { name: /Continue session|Review session/ }).click();
    await expect(page).toHaveURL(/\/lesson$/);
    await expect(page.getByRole('link', { name: /Start knowledge check/ })).toBeVisible();
  } finally {
    await assertBrowserClean();
  }
});

test('every Phase 1 route has intentional visible content on direct navigation', async ({
  page,
}, testInfo) => {
  const assertBrowserClean = observeBrowser(page, testInfo);
  const routes = [
    ['/today', /Good evening|session is unavailable|Nothing scheduled/],
    ['/lesson', /Reference architecture|Lesson unavailable|No lesson is ready/],
    ['/quiz', /Question 1 of|Knowledge check unavailable|No questions in this session/],
    ['/progress', /Progress that compounds/],
    ['/library', /Teach Nora what matters to you/],
    ['/interview', /Think like a principal engineer/],
    ['/preferences', /Learning preferences|Preferences unavailable|No preferences were returned/],
  ] as const;
  try {
    for (const [route, content] of routes) {
      await page.goto(route);
      await expect(page.getByText(content).first()).toBeVisible();
    }
    await page.reload();
    await expect(page.locator('main')).not.toBeEmpty();
  } finally {
    await assertBrowserClean();
  }
});

test('quiz completion is persisted before navigation to progress', async ({ page }, testInfo) => {
  test.skip(!demoMode, 'Deterministic answer fixtures are available in demo mode.');
  const assertBrowserClean = observeBrowser(page, testInfo);
  try {
    await page.goto('/quiz');
    const total = Number(
      (await page.getByText(/Question 1 of \d+/).textContent())?.match(/of (\d+)/)?.[1]
    );
    for (let index = 0; index < total; index++) {
      const options = page.locator('.answer-options button');
      if (await options.count()) await options.first().click();
      else
        await page
          .locator('.scenario-answer textarea')
          .fill('A reasoned answer with explicit trade-offs.');
      await page.getByRole('button', { name: /Check answer/ }).click();
      await page
        .getByRole('button', {
          name: index === total - 1 ? /Submit & see results/ : /Next question/,
        })
        .click();
    }
    await expect(page.getByRole('heading', { name: /You nailed it|Good work/ })).toBeVisible();
    await page.getByRole('button', { name: /Finish session/ }).click();
    await expect(page).toHaveURL(/\/progress$/);
    await expect(page.getByRole('heading', { name: 'Progress that compounds' })).toBeVisible();
  } finally {
    await assertBrowserClean();
  }
});

test('document upload and preference save expose bounded action feedback', async ({
  page,
}, testInfo) => {
  test.skip(!demoMode, 'Mutation isolation uses the in-browser demo adapter.');
  const assertBrowserClean = observeBrowser(page, testInfo);
  try {
    await page.goto('/library');
    await page.locator('input[type=file]').setInputFiles({
      name: 'reliability-notes.md',
      mimeType: 'text/markdown',
      buffer: Buffer.from('# Reliability\nPrefer explicit failure states.'),
    });
    await expect(page.getByText('reliability-notes.md')).toBeVisible();
    await page.goto('/preferences');
    await page.getByLabel('Coaching mode').selectOption('Mentor');
    await page.getByRole('button', { name: 'Save preferences' }).click();
    await expect(page.getByText('Preferences saved')).toBeVisible();
  } finally {
    await assertBrowserClean();
  }
});

test('interview does not send until connected and produces a scorecard', async ({
  page,
}, testInfo) => {
  test.skip(!demoMode, 'Deterministic interview prompts are available in demo mode.');
  const assertBrowserClean = observeBrowser(page, testInfo);
  try {
    await page.goto('/interview');
    await page.getByRole('button', { name: /Enter interview room/ }).click();
    const answer = page.locator('.message-box textarea');
    await expect(answer).toBeEnabled();
    await answer.fill('Answer 1: assumptions, trade-offs, and recovery plan.');
    await page.locator('.message-box').evaluate((form) => {
      form.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }));
      form.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }));
    });
    await expect(page.locator('.conversation article.you')).toHaveCount(1);
    await expect(answer).toBeDisabled();
    await expect(page.getByText('Answer sent. Waiting for the interviewer…')).toBeVisible();

    for (let turn = 1; turn < 5; turn++) {
      await expect(answer).toBeEnabled();
      await answer.fill(`Answer ${turn + 1}: assumptions, trade-offs, and recovery plan.`);
      await page.getByRole('button', { name: /Send answer/ }).click();
    }
    await expect(page.getByText('INTERVIEW COMPLETE')).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Your scorecard is ready.' })).toBeVisible();
  } finally {
    await assertBrowserClean();
  }
});

test('daily session failure is visible and retryable without a blank application', async ({
  page,
}, testInfo) => {
  test.skip(demoMode, 'Response interception exercises the real HTTP adapter.');
  const assertBrowserClean = observeBrowser(page, testInfo, [/\/sessions\/daily.*HTTP 503/]);
  try {
    await page.route('**/api/v1/sessions/daily', (route) =>
      route.fulfill({
        status: 503,
        contentType: 'application/problem+json',
        body: JSON.stringify({
          title: 'Session service unavailable',
          detail: 'Try again shortly.',
        }),
      })
    );
    await page.goto('/today');
    await expect(
      page.getByRole('heading', { name: 'Today’s session is unavailable' })
    ).toBeVisible();
    await expect(page.getByRole('button', { name: 'Try again' })).toBeVisible();
    await expect(page.locator('nav')).toBeVisible();
  } finally {
    await assertBrowserClean();
  }
});
