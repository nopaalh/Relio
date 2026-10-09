import { test, expect } from "@playwright/test";
test("summary shows real source ACV and filters",async({page})=>{
  await page.goto("/deals");await expect(page.getByRole("heading",{name:"See the universe behind every deal."})).toBeVisible();
  await expect(page.locator(".metric-value.money")).toHaveText(/667.800.000/);
  await expect(page.locator(".deal-table tbody tr")).toHaveCount(5);
  await page.screenshot({path:"../docs/frontend/qa/relio-summary-1440.png",fullPage:true});
  await page.getByRole("textbox",{name:"Search deals"}).fill("Nirwana");await expect(page.locator(".deal-table tbody tr")).toHaveCount(1);
  await page.getByRole("textbox",{name:"Search deals"}).fill("tidak-ada");await expect(page.getByRole("heading",{name:"No matching deals"})).toBeVisible();
  await page.getByRole("button",{name:"Reset filters",exact:true}).click();await expect(page.locator(".deal-table tbody tr")).toHaveCount(5);
});
test("timeline selection, source provenance, graph list and focus",async({page})=>{
  await page.goto("/deals/DL-004");await expect(page.getByRole("heading",{name:"Nirwana Hotel & Resto"})).toBeVisible();
  await page.getByRole("button",{name:"Open evidence",exact:true}).click();await expect(page.getByRole("dialog").locator("blockquote")).toContainText("Kami tunda dulu");await page.keyboard.press("Escape");
  await page.getByRole("button",{name:/Discovery Nirwana/}).click();await page.getByRole("button",{name:"Open evidence",exact:true}).click();await expect(page.getByRole("dialog").locator("blockquote")).toContainText("35 outlet");await page.keyboard.press("Escape");
  await page.getByLabel("Universe view options").click();await page.getByRole("button",{name:"Connections",exact:true}).click();await expect(page.locator(".relation-row")).not.toHaveCount(0);
  await page.getByRole("button",{name:"View universe",exact:true}).click();
  await expect(page.locator(".react-flow__node")).not.toHaveCount(0);
  await page.getByRole("button",{name:"Focus selected event"}).click();await page.keyboard.press("Escape");
  await expect(page.getByRole("button",{name:"All context",exact:true})).toBeVisible();
});
test("historical as-of hides future evidence and snapshot facts",async({page,request})=>{
  await page.goto("/deals/DL-004?as_of=2026-09-21");
  await expect(page.getByRole("heading",{name:"Nirwana Hotel & Resto"})).toBeVisible();
  await expect(page.locator(".timeline-event")).toHaveCount(2);
  await expect(page.locator(".timeline-list")).not.toContainText("Procurement update");
  await expect(page.locator(".detail-title .badge")).toHaveCount(0);
  const future=await request.get("/api/evidence/EVI-I0335?as_of=2026-09-21");expect(future.status()).toBe(404);
  const historical=await request.get("/api/deals/DL-004?as_of=2026-09-21");const body=await historical.json();expect(body.deal.acv).toBeNull();expect(body.deal.stage).toBeNull();
  await page.getByRole("button",{name:"Next event",exact:true}).click();await expect(page.locator(".timeline-list")).toContainText("Procurement update");
});
test("two sourced actions, unavailable JEV, stale results, precedent evidence",async({page,request})=>{
  await page.goto("/deals/DL-004");await expect(page.locator(".candidate")).toHaveCount(4);
  const run=page.getByRole("button",{name:"Compare approaches",exact:true});await expect(run).toBeDisabled();
  await page.locator(".candidate input").nth(0).check();await expect(run).toBeDisabled();await page.locator(".candidate input").nth(1).check();await expect(run).toBeEnabled();await run.click();
  await expect(page.locator(".comparison-result")).toContainText("JEV not connected");await expect(page.locator(".result-card")).toHaveCount(2);await expect(page.locator(".comparison-result")).toContainText("No scores or rankings");
  await page.locator(".candidate input").nth(2).check();await expect(page.locator(".comparison-result")).toHaveCount(0);
  await expect(page.getByText("Selection changed.",{exact:false})).toBeVisible();
  await page.locator(".candidate").first().getByRole("button",{name:/D-2025-06/}).click();
  await expect(page.getByRole("dialog")).toBeVisible();await expect(page.getByRole("dialog")).toContainText("decision_log.csv");await page.keyboard.press("Escape");
  const duplicate=await request.post("/api/deals/DL-004/actions/compare",{data:{as_of:"2026-10-01",action_ids:["HIST-PILOT-STARTER","HIST-PILOT-STARTER"]}});expect(duplicate.status()).toBe(400);
});
test("Copilot reports provider unavailable and preserves original scope",async({page})=>{
  await page.goto("/deals/DL-004");await page.getByRole("button",{name:"Copilot",exact:true}).click();
  await page.getByRole("textbox",{name:"Question for Copilot"}).fill("Apa hambatan utama?");await page.getByRole("button",{name:"Send question"}).click();
  await expect(page.locator(".assistant-message")).toContainText("Copilot is not connected");
  await page.getByLabel("Context date").fill("2026-09-21");await expect(page.locator(".user-message")).toContainText("2026-10-01");
});
for(const width of [375,768,1024,1440])test("responsive layout "+width,async({page})=>{
  await page.setViewportSize({width,height:1000});await page.goto("/deals/DL-004");await expect(page.locator(".candidate")).toHaveCount(4);
  expect(await page.evaluate(()=>document.documentElement.scrollWidth<=window.innerWidth+1)).toBe(true);
  if(width<=900){await page.getByRole("tab",{name:"Timeline",exact:true}).click();await expect(page.locator(".timeline-panel")).toBeVisible();await page.getByRole("tab",{name:"Evidence",exact:true}).click();await expect(page.locator(".evidence-panel")).toBeVisible();}
  await page.getByRole("tab",{name:"Graph",exact:true}).isVisible().then(async visible=>{if(visible)await page.getByRole("tab",{name:"Graph",exact:true}).click();});
  await expect(page.locator(".react-flow__node")).not.toHaveCount(0);
  await expect.poll(async()=>page.locator(".react-flow__node").first().evaluate(el=>{const b=el.getBoundingClientRect(),c=el.closest(".graph-container")!.getBoundingClientRect();return b.right>c.left&&b.left<c.right&&b.bottom>c.top&&b.top<c.bottom;})).toBe(true);
  await page.screenshot({path:"../docs/frontend/qa/relio-detail-"+width+".png",fullPage:true});
  if(width===375)await page.locator(".graph-panel").screenshot({path:"../docs/frontend/qa/relio-graph-375.png"});
});
test("unknown deal and invalid date fail explicitly",async({request,page})=>{
  expect((await request.get("/api/deals?as_of=2026-02-30")).status()).toBe(400);
  expect((await request.get("/api/deals/no-such-deal")).status()).toBe(404);
  await page.goto("/deals/no-such-deal");await expect(page.getByRole("heading",{name:"Context could not be loaded"})).toBeVisible();
});
