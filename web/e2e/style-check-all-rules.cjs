const { chromium } = require('@playwright/test');

(async () => {
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();
  await page.goto('http://localhost:5173', { waitUntil: 'networkidle', timeout: 15000 });
  await page.waitForTimeout(1500);

  const allRules = await page.evaluate(() => {
    const results = [];
    for (const sheet of document.styleSheets) {
      try {
        for (const rule of sheet.cssRules) {
          if (rule.selectorText) {
            results.push(rule.selectorText);
          }
        }
      } catch(e) {}
    }
    return results;
  });

  // Search for bg-, text-, border-, rounded- utility classes
  const bgRules = allRules.filter(r => r && r.match(/\.bg-/));
  const roundedRules = allRules.filter(r => r && r.includes('rounded-'));
  const cardRules2 = allRules.filter(r => r && r.includes('card'));
  
  console.log(`Total CSS rules found: ${allRules.length}`);
  console.log(`bg-* utilities found: ${bgRules.length}`);
  console.log('Sample bg-* rules:', bgRules.slice(0, 5));
  console.log(`rounded-* utilities found: ${roundedRules.length}`);
  console.log('Sample rounded-* rules:', roundedRules.slice(0, 5));
  console.log('card-related rules:', cardRules2);

  // Check body background
  const bodyBg = await page.evaluate(() => getComputedStyle(document.body).backgroundColor);
  console.log('Body background:', bodyBg);

  await browser.close();
})();
