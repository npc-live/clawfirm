import { writeFileSync, mkdirSync, existsSync, readdirSync, unlinkSync } from "node:fs";
import { homedir } from "node:os";
import { join } from "node:path";
import { BASE_URL } from "./auth.js";

const SKILLS_DIR = join(homedir(), ".skillctl", "skills");

const GREEN = "\x1b[32m";
const BLUE = "\x1b[34m";
const RED = "\x1b[31m";
const NC = "\x1b[0m";

const ok = (msg) => console.log(`${GREEN}  ✓${NC}  ${msg}`);
const info = (msg) => console.log(`${BLUE}  →${NC}  ${msg}`);
const fail = (msg) => console.log(`${RED}  ✗${NC}  ${msg}`);

export async function installSkills(session) {
  info("Fetching skills...");

  let skills;
  try {
    const res = await fetch(`${BASE_URL}/api/skills`, {
      headers: { Cookie: session.cookie },
    });
    if (!res.ok) {
      if (res.status !== 404) fail(`Failed to fetch skills: ${res.statusText}`);
      return false;
    }
    skills = await res.json(); // [{ name: string, content: string }]
  } catch (e) {
    fail(`Network error: ${e.message}`);
    return false;
  }

  if (!existsSync(SKILLS_DIR)) mkdirSync(SKILLS_DIR, { recursive: true });

  for (const skill of skills) {
    writeFileSync(join(SKILLS_DIR, `${skill.name}.md`), skill.content);
  }

  ok(`${skills.length} skill(s) installed`);
  return true;
}

export function removeSkills() {
  if (!existsSync(SKILLS_DIR)) return;
  const files = readdirSync(SKILLS_DIR).filter((f) => f.endsWith(".md"));
  for (const f of files) unlinkSync(join(SKILLS_DIR, f));
}
