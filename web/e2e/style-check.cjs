const { chromium } = require('@playwright/test');

(async () => {
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();

  await page.goto('http://localhost:5173', { waitUntil: 'networkidle', timeout: 15000 });
  await page.waitForTimeout(1000);

  const cardStyles = await page.evaluate(() => {
    const card = document.querySelector('.bg-card');
    if (!card) return { found: false, error: 'No .bg-card element found' };
    const computed = getComputedStyle(card);
    return {
      found: true,
      backgroundColor: computed.backgroundColor,
      borderRadius: computed.borderRadius,
      padding: computed.padding,
      border: computed.border,
    };
  });
  console.log('Card styles:', JSON.stringify(cardStyles, null, 2));

  const btnStyles = await page.evaluate(() => {
    const btn = document.querySelector('button');
    if (!btn) return { found: false };
    const computed = getComputedStyle(btn);
    return {
      found: true,
      backgroundColor: computed.backgroundColor,
      color: computed.color,
      height: computed.height,
      fontSize: computed.fontSize,
      borderRadius: computed.borderRadius,
    };
  });
  console.log('Button styles:', JSON.stringify(btnStyles, null, 2));

  const bodyStyles = await page.evaluate(() => {
    const computed = getComputedStyle(document.body);
    return {
      backgroundColor: computed.backgroundColor,
      color: computed.color,
      fontSize: computed.fontSize,
      fontFamily: computed.fontFamily,
    };
  });
  console.log('Body styles:', JSON.stringify(bodyStyles, null, 2));

  const tabularNums = await page.evaluate(() => {
    return getComputedStyle(document.body).fontVariantNumeric;
  });
  console.log('Body font-variant-numeric:', tabularNums);

  await browser.close();
})();
