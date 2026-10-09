import { defineConfig } from "@playwright/test";
import { existsSync, readdirSync } from "node:fs";
import path from "node:path";
import os from "node:os";
function browserPath():string|undefined {
  if(process.env.RELIO_BROWSER_PATH)return process.env.RELIO_BROWSER_PATH;
  const cache=path.join(os.homedir(),".cache","puppeteer","chrome-headless-shell");
  if(existsSync(cache)){
    for(const version of readdirSync(cache).sort((a,b)=>b.localeCompare(a,undefined,{numeric:true}))){
      const executable=path.join(cache,version,process.platform==="win32"?"chrome-headless-shell-win64/chrome-headless-shell.exe":"chrome-headless-shell-linux64/chrome-headless-shell");
      if(existsSync(executable))return executable;
    }
  }
  const chrome=process.platform==="win32"?"C:/Program Files/Google/Chrome/Application/chrome.exe":"/usr/bin/chromium";
  return existsSync(chrome)?chrome:undefined;
}
const baseURL=process.env.RELIO_TEST_BASE_URL??"http://127.0.0.1:3000";
export default defineConfig({
  testDir:"./tests/e2e",fullyParallel:false,workers:1,timeout:60000,
  reporter:[["list"],["html",{open:"never"}]],
  use:{baseURL,headless:true,viewport:{width:1440,height:1000},screenshot:"only-on-failure",
    launchOptions:{executablePath:browserPath(),args:["--disable-gpu"]}},
  webServer:{command:"npm run dev",url:baseURL,reuseExistingServer:true,timeout:120000,env:{RELIO_DATA_MODE:"local"}}
});
