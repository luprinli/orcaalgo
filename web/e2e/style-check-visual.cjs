const { chromium } = require('@playwright/test');

(async () => {
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();
  await page.goto('http://localhost:5173', { waitUntil: 'networkidle', timeout: 15000 });
  await page.waitForTimeout(1500);

  // Take a screenshot
  await page.screenshot({ path: 'e2e/login-page-screenshot.png', fullPage: false });

  // Re-check all styles with different approach: inline style elements
  const styleCount = await page.evaluate(() => {
    const styleEls = document.querySelectorAll('style');
    let result = [];
    styleEls.forEach(el => {
      const hasBgCard = el.textContent.includes('bg-card');
      result.push({ 
        length: el.textContent.length, 
        hasBgCard,
        prefix: el.textContent.slice(0, 100)
      });
    });
    return result;
  });
  console.log('Inline style elements:', JSON.stringify(styleCount, null, 2));

  // Check the number of <link> stylesheets
  const linkCount = await page.evaluate(() => {
    const links = document.querySelectorAll('link[rel="stylesheet"]');
    return links.length;
  });
  console.log('Link stylesheet count:', linkCount);

  // Direct check: does the card element have visual background?
  const visualCheck = await page.evaluate(() => {
    const card = document.querySelector('.bg-card');
    if (!card) return { found: false };
    const rect = card.getBoundingClientRect();
    return {
      found: true,
      rect: { x: rect.x, y: rect.y, width: rect.width, height: rect.height },
      classList: Array.from(card.classList),
      computedStyles: {
        bg: getComputedStyle(card).backgroundColor,
        radius: getComputedStyle(card).borderRadius,
        padding: getComputedStyle(card).padding,
        border: getComputedStyle(card).border,
      }
    };
  });
  console.log('Visual check:', JSON.stringify(visualCheck, null, 2));

  await browser.close();
})();
