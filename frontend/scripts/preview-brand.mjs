import fs from 'node:fs/promises';
import path from 'node:path';
import os from 'node:os';
import {chromium} from '@playwright/test';
const folder=path.resolve('../Relio_GSM_Universe_v3/previews');
const svg=await fs.readFile(path.join(folder,'Relio_Universe_Brand_Board.svg'),'utf8');
const fonts=await Promise.all([['Inter',400,'inter'],['Manrope',600,'manrope']].map(async([family,weight,slug])=>{
  const data=await fs.readFile('node_modules/@fontsource/'+slug+'/files/'+slug+'-latin-'+weight+'-normal.woff2');
  return '@font-face{font-family:'+family+';font-weight:'+weight+';src:url(data:font/woff2;base64,'+data.toString('base64')+')}';
}));
let executablePath=process.env.RELIO_BROWSER_PATH;
if(!executablePath){
 const cache=path.join(os.homedir(),'.cache/puppeteer/chrome-headless-shell');
 for(const version of (await fs.readdir(cache).catch(()=>[])).sort().reverse()){
  const candidate=path.join(cache,version,process.platform==='win32'?'chrome-headless-shell-win64/chrome-headless-shell.exe':'chrome-headless-shell-linux64/chrome-headless-shell');
  if(await fs.access(candidate).then(()=>true,()=>false)){executablePath=candidate;break}
 }
}
const browser=await chromium.launch({headless:true,executablePath});
try{
  const page=await browser.newPage({viewport:{width:1400,height:960}});
  await page.setContent('<html><style>'+fonts.join('')+'body{margin:0}body>svg{display:block;width:1400px;height:960px}</style><body>'+svg+'</body></html>');
  await page.evaluate(()=>document.fonts.ready);
  const xml=await page.evaluate(s=>new DOMParser().parseFromString(s,'image/svg+xml').querySelector('parsererror')?.textContent??null,svg);
  if(xml)throw new Error(xml);
  await page.locator('body>svg').screenshot({path:path.join(folder,'Relio_Universe_Brand_Board.png')});
  console.log('Brand board rendered; SVG XML valid.');
}finally{await browser.close()}
