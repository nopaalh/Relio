import {test,expect} from '@playwright/test';
test('English VP Sales navigation and functional evidence library',async({page})=>{
 await page.goto('/deals');await expect(page.getByRole('heading',{name:'See the universe behind every deal.'})).toBeVisible();
 await expect(page.locator('html')).toHaveAttribute('lang','en');await expect(page.locator('.profile')).toContainText('Andi Wiratama');await expect(page.locator('.profile')).toContainText('VP Sales');
 for(const name of ['Dashboard','Deal Universe','Approaches'])await expect(page.getByRole('navigation').getByRole('link',{name,exact:true})).toBeVisible();
 await page.getByRole('button',{name:'Andi Wiratama profile'}).click();await expect(page.getByRole('dialog')).toContainText('andi@kasirnusa.id');await page.keyboard.press('Escape');
 await page.getByRole('button',{name:'Evidence library'}).click();await expect(page.getByRole('dialog').locator('.source-link')).not.toHaveCount(0);await page.getByRole('dialog').locator('.source-link').first().click();await expect(page.getByRole('dialog')).toContainText('Full source record');await page.keyboard.press('Escape');
});
test('dates sit above planets, history search and scrub preserve as-of boundaries',async({page,request})=>{
 await page.setViewportSize({width:375,height:950});await page.goto('/deals/DL-004');await expect(page.locator('.react-flow__node')).toHaveCount(6);
 for(const box of await page.locator('.universe-navigator .icon-button').evaluateAll(es=>es.map(e=>({width:e.getBoundingClientRect().width,height:e.getBoundingClientRect().height})))){expect(box.width).toBeGreaterThanOrEqual(44);expect(box.height).toBeGreaterThanOrEqual(44)}
 const event=page.locator('.react-flow__node').filter({hasText:'Procurement update'});await expect(event.locator('time')).toHaveAttribute('datetime','2026-09-22');
 expect(await event.evaluate(el=>el.querySelector('time')!.getBoundingClientRect().bottom<el.querySelector('.planet-orb')!.getBoundingClientRect().top)).toBe(true);
 await expect(page.locator('.react-flow__node').filter({hasText:'P04 · Deal'}).locator('time')).toContainText('As of');
 await page.getByLabel('Search universe dates').fill('2026-09-22');await expect(page.locator('.date-rail button')).toHaveCount(1);await page.locator('.date-rail button').click();await expect(page.getByLabel('Context date')).toHaveValue('2026-09-22');
 const range=page.getByLabel('Scrub universe history');await range.evaluate((el:HTMLInputElement)=>{Object.getOwnPropertyDescriptor(HTMLInputElement.prototype,'value')!.set!.call(el,String(Date.parse('2026-09-21')/86400000));el.dispatchEvent(new Event('input',{bubbles:true}));el.dispatchEvent(new Event('change',{bubbles:true}));});
 await expect(page.getByLabel('Context date')).toHaveValue('2026-09-21');await expect(page.locator('.timeline-event')).toHaveCount(2);await expect(page.locator('.graph-container')).not.toContainText('I0335');
 expect((await request.get('/api/evidence/EVI-I0335?as_of=2026-09-21')).status()).toBe(404);
 await page.getByLabel('Search universe dates').fill('');await page.getByRole('button',{name:'Expand history'}).click();await expect(page.locator('.universe-navigator')).toHaveClass(/expanded/);
});
test('search planet opens popup and contextual Copilot retains original message scope',async({page})=>{
 await page.goto('/deals/DL-004');await page.getByLabel('Search planets').fill('EV-I0335');await page.locator('.planet-search-results button').click();
 const popup=page.getByRole('dialog',{name:'Context Explorer'});await expect(popup).toBeVisible();await expect(popup).toContainText('Connected to');await expect(page.locator('#bottom-copilot')).toHaveCount(0);
 await popup.getByRole('button',{name:'Ask Copilot',exact:true}).click();await expect(popup).not.toBeVisible();const drawer=page.getByRole('dialog',{name:'Copilot',exact:true});await expect(drawer).toBeVisible();await expect(drawer.locator('.copilot-entity')).toContainText('EV-I0335');await expect(drawer.locator('.copilot-service-note')).toContainText('answer provider is not connected');
 await drawer.getByRole('textbox',{name:'Question for Copilot'}).fill('What is this request?');const sent=page.waitForRequest(r=>r.url().endsWith('/api/copilot/ask'));
 await drawer.getByRole('button',{name:'Send question'}).click();const payload=(await sent).postDataJSON();expect(payload.question).toContain('EV-I0335');expect(payload.as_of).toBe('2026-10-01');expect(payload.deal_id).toBe('DL-004');
 await expect(drawer.locator('.assistant-message')).toContainText('Copilot is not connected');await page.getByLabel('Context date').fill('2026-09-21');await expect(drawer.locator('.user-message')).toContainText('2026-10-01');await expect(drawer.locator('.message-entity')).toContainText('EV-I0335');await expect(drawer.locator('.copilot-entity')).toHaveCount(0);
 await drawer.screenshot({path:'../docs/frontend/qa/relio-vp-copilot-1440.png'});await drawer.getByRole('button',{name:'Close Copilot'}).click();
});
test('mobile contextual Copilot replaces node popup and closes with Escape',async({page})=>{
 await page.setViewportSize({width:375,height:950});await page.goto('/deals/DL-004');await page.getByLabel('Search planets').fill('I0335');await page.locator('.planet-search-results button').filter({hasText:'Procurement update'}).click();
 await page.getByRole('dialog',{name:'Context Explorer'}).getByRole('button',{name:'Ask Copilot',exact:true}).click();const drawer=page.getByRole('dialog',{name:'Copilot',exact:true});await expect(drawer).toBeVisible();await drawer.screenshot({path:'../docs/frontend/qa/relio-vp-copilot-375.png'});await page.keyboard.press('Escape');await expect(drawer).not.toBeVisible();
 expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1)).toBe(true);
});
