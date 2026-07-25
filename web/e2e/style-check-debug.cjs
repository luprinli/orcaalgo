const { chromium } = require('@playwright/test');

(async () => {
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();

  await page.goto('http://localhost:5173', { waitUntil: 'networkidle', timeout: 15000 });
  await page.waitForTimeout(1000);

  // Check what elements match .bg-card, and their class list + parent
  const debug = await page.evaluate(() => {
    const cards = document.querySelectorAll('.bg-card');
    const results = [];
    cards.forEach((el, i) => {
      results.push({
        index: i,
        tagName: el.tagName,
        className: el.className,
        parentTag: el.parentElement?.tagName,
        parentClass: el.parentElement?.className?.slice(0, 200),
        computedBg: getComputedStyle(el).backgroundColor,
        computedRadius: getComputedStyle(el).borderRadius,
        computedPadding: getComputedStyle(el).padding,
        computedBorder: getComputedStyle(el).border,
      });
    });
    return results;
  });
  console.log('All .bg-card elements:', JSON.stringify(debug, null, 2));

  // Check if there's a Card component element by looking for rounded-lg class
  const cardDebug = await page.evaluate(() => {
    const els = document.querySelectorAll('.rounded-lg');
    const results = [];
    els.forEach((el, i) => {
      if (i >= 5) return;
      results.push({
        index: i,
        tagName: el.tagName,
        className: el.className,
        computedBg: getComputedStyle(el).backgroundColor,
        computedRadius: getComputedStyle(el).borderRadius,
        computedPadding: getComputedStyle(el).padding,
      });
    });
    return results;
  });
  console.log('First .rounded-lg elements:', JSON.stringify(cardDebug, null, 2));

  // Check all computed CSS custom properties on the root
  const cssVars = await page.evaluate(() => {
    const styles = getComputedStyle(document.documentElement);
    const vars = ['--tw-bg', '--tw-card', '--tw-fg', '--tw-border', '--tw-radius', '--bg-card'];
    const result = {};
    vars.forEach(v => { result[v] = styles.getPropertyValue(v).trim(); });
    return result;
  });
  console.log('CSS custom properties:', JSON.stringify(cssVars, null, 2));

  // Check if .bg-card CSS rule exists in stylesheets
  const hasRule = await page.evaluate(() => {
    for (const sheet of document.styleSheets) {
      try {
        for (const rule of sheet.cssRules) {
          if (rule.selectorText && rule.selectorText.includes('bg-card')) {
            return { found: true, text: rule.cssText.slice(0, 300) };
          }
        }
      } catch (e) { /* cross-origin sheet */ }
    }
    return { found: false };
  });
  console.log('CSS rule for bg-card:', JSON.stringify(hasRule, null, 2));

  await browser.close();
})();
