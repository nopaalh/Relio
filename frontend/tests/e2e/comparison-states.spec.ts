import { test, expect } from "@playwright/test";
// Isolated browser fixtures exercise unavailable/partial/disagreement UI.
// These values are DATA UJI only and never enter the application dataset.
test("DATA UJI: separate Score/Choice, disagreement, invalid Choice, partial score",async({page})=>{
  let partial=false;
  await page.route("**/api/deals/DL-004/actions/compare",async route=>{
    const body=route.request().postDataJSON();
    await route.fulfill({json:{deal_id:"DL-004",as_of:body.as_of,comparison_type:"action_suitability_not_win_probability",assessment_status:partial?"partial":"complete",rubric_version:"DATA-UJI",
      items:body.action_ids.map((id:string,index:number)=>({action_id:id,suitability_score_100:partial&&index===1?null:index===0?75:50,jev_raw_score:partial&&index===1?null:index===0?3:2,jev_confidence:null,jev_choice_preference:partial?.8:index===0?.2:.8,rank:index+1,evidence_ids:[],precedent_ids:[],policy_flags:[],unknowns:["DATA UJI · bukan hasil provider asli"]})),
      warning:"DATA UJI. Kesesuaian bukan peluang menang."}});
  });
  await page.goto("/deals/DL-004");
  await page.locator(".candidate input").nth(0).check();await page.locator(".candidate input").nth(1).check();
  await page.getByRole("button",{name:"Compare approaches",exact:true}).click();
  await expect(page.locator(".comparison-result")).toContainText("Review needed");
  await expect(page.locator(".result-card").first()).toContainText("20.0%");
  partial=true;await page.getByRole("button",{name:"Compare approaches",exact:true}).click();
  await expect(page.locator(".comparison-result")).toContainText("partial");
  await expect(page.locator(".result-card").nth(1).locator(".score-value")).toHaveText(/—/);
  await expect(page.locator(".result-card").first()).toContainText("Choice · selected setUnavailable");
  await expect(page.locator(".result-card").first()).toContainText("RankUnavailable");
});
test("keyboard access, dialog Escape and restored focus",async({page})=>{
  await page.goto("/deals/DL-004");
  const trigger=page.getByRole("button",{name:/Attractiveness.*Why this score/});await trigger.focus();await page.keyboard.press("Enter");
  await expect(page.getByRole("dialog")).toBeVisible();await page.keyboard.press("Escape");await expect(trigger).toBeFocused();
  await page.getByLabel("Universe view options").click();await page.getByRole("button",{name:"Connections",exact:true}).click();
  const event=page.locator(".relation-row").filter({hasText:"Procurement update"}).first();await event.focus();await page.keyboard.press("Enter");
  await expect(page.locator(".context-inspector")).toContainText("Connected to");
  await page.getByRole("tab",{name:"Source evidence",exact:true}).click();await expect(page.getByRole("dialog").locator(".evidence-content blockquote")).toContainText("Kami tunda dulu");await page.keyboard.press("Escape");
  const option=page.locator(".candidate input").first();await option.focus();await page.keyboard.press("Space");await expect(option).toBeChecked();
});
