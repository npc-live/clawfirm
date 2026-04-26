#!/usr/bin/env tsx
// Run this once to generate your MASTER_ENCRYPTED + MASTER_SALT values
// Usage: pnpm setup
// Then copy the output into your .env file

import { encryptWithPassword } from "../src/crypto.ts"
import { randomBytes } from "crypto"

const run = async () => {
  // Generate a random master key (this is the inner key that encrypts wallet private keys)
  const masterKey = randomBytes(32).toString("base64url")

  const password = process.env.MASTER_ENCRYPTION_PASSWORD_CRYPTO
  if (!password) {
    console.error("Error: Set MASTER_ENCRYPTION_PASSWORD_CRYPTO in your environment first")
    console.error("Example: MASTER_ENCRYPTION_PASSWORD_CRYPTO=my-password pnpm setup")
    process.exit(1)
  }

  console.log("Generating master key...\n")
  const { encrypted, salt } = await encryptWithPassword(masterKey, password)

  console.log("Add these to your .env file:\n")
  console.log(`MASTER_ENCRYPTION_PASSWORD_CRYPTO=${password}`)
  console.log(`MASTER_ENCRYPTED=${encrypted}`)
  console.log(`MASTER_SALT=${salt}`)
  console.log()
  console.log("Keep MASTER_ENCRYPTION_PASSWORD_CRYPTO secret — it's the root of your security.")
  console.log("MASTER_ENCRYPTED + MASTER_SALT are safe to store (useless without the password).")
}

run().catch(err => {
  console.error("Setup failed:", err instanceof Error ? err.message : err)
  process.exit(1)
})
