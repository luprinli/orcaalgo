const { chromium } = require('@playwright/test');

(async () => {
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();
  await page.goto('http://localhost:5173', { waitUntil: 'networkidle', timeout: 15000 });
  await page.waitForTimeout(1000);

  // Enumerate all CSS rules referencing card, bg, or background
  const cardRules = await page.evaluate(() => {
    const results = [];
    for (const sheet of document.styleSheets) {
      try {
        for (const rule of sheet.cssRules) {
          const text = rule.cssText || '';
          if (text.includes('card') || (rule.selectorText && rule.selectorText.includes('bg-'))) {
            results.push(text.slice(0, 200));
          }
        }
      } catch(e) {}
    }
    return results;
  });
  console.log('CSS rules with "card" or "bg-":', JSON.stringify(cardRules, null, 2));

  // Now check the actual card element more deeply
  const cardDebug = await page.evaluate(() => {
    const card = document.querySelector('.bg-card');
    if (!card) return { found: false };
    const styles = getComputedStyle(card);
    // Check if --tw-card is defined on this element
    const inlineBg = card.style.backgroundColor;
    // Check all stylesheets for class
    const classList = Array.from(card.classList);
    return {
      found: true,
      classList,
      backgroundColor: styles.backgroundColor,
      bgImage: styles.backgroundImage,
      // Check if the element inherits any background
      parentBg: getComputedStyle(card.parentElement).backgroundColor,
      // Check all matched CSS rules for this element
    };
  });
  console.log('Card element debug:', JSON.stringify(cardDebug, null, 2));

  await browser.close();
})();
