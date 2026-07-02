export const meta = {
  name: 'scan-and-fix',
  description: 'Fix hardcoded colors and spacing across all frontend pages',
  phases: [
    { title: 'Read-&-Fix', detail: 'One agent per file' },
    { title: 'Verify', detail: 'Build + test' },
  ],
};

var files = [
  "frontend-next/src/app/(main)/owner/page.tsx",
  "frontend-next/src/app/(main)/settings/page.tsx",
  "frontend-next/src/app/(main)/batch-ops/page.tsx",
  "frontend-next/src/app/(main)/products/dashboard/page.tsx",
  "frontend-next/src/app/(main)/suppliers/page.tsx",
  "frontend-next/src/app/(main)/settings/policy/page.tsx",
  "frontend-next/src/app/(main)/image-gen/canvas/page.tsx",
  "frontend-next/src/app/(main)/error.tsx",
  "frontend-next/src/app/(main)/candidates/page.tsx",
  "frontend-next/src/app/(main)/products/[id]/page.tsx",
  "frontend-next/src/app/(main)/products/[id]/suppliers/page.tsx",
  "frontend-next/src/app/(main)/products/[id]/compliance-tab.tsx",
];

var FIX_PROMPT = [
  "Read this file then fix ALL of these issues:",
  "",
  "1. Replace ALL hardcoded hex colors with CSS variables:",
  "   #999 -> var(--t3)",
  "   #666 -> var(--t3)",
  "   #52c41a -> var(--g4)",
  "   #faad14 -> var(--y4)",
  "   #1677ff -> var(--i4)",
  "   #722ed1 -> var(--i5)",
  "   #13c2c2 -> var(--c4)",
  "   #fa8c16 -> var(--y4)",
  "   #f5222d -> var(--r4)",
  "   #f0f5ff -> var(--s2)",
  "   #ff4d4f -> var(--r4)",
  "   #f6ffed -> var(--s1)",
  "   #b7eb8f -> var(--g4)",
  "   #cf1322 -> var(--r4)",
  "   #d9d9d9 -> var(--t4)",
  "",
  "2. Replace hardcoded spacing with CSS variables:",
  "   fontSize 12 -> 'var(--text-small)'",
  "   fontSize 13 -> 'var(--text-body)'",
  "   padding 24 -> 'var(--space-xl)'",
  "   padding 48 -> 'var(--space-3xl)'",
  "   padding 64 -> 'var(--space-3xl)'",
  "   marginBottom 8 -> 'var(--space-sm)'",
  "   marginBottom 12 -> 'var(--space-md)'",
  "   marginBottom 16 -> 'var(--space-lg)'",
  "   marginBottom 20 -> 'var(--space-lg)'",
  "   marginBottom 24 -> 'var(--space-xl)'",
  "   marginTop 4 -> 'var(--space-xs)'",
  "   marginTop 8 -> 'var(--space-sm)'",
  "   marginTop 12 -> 'var(--space-md)'",
  "   marginTop 16 -> 'var(--space-lg)'",
  "",
  "3. For owner/page.tsx: also replace repeating stat card divs (div with bg/border/radius/padding 8 times) with <StatCard> component imported from '@/components/ui/StatCard'. Also replace section container divs with <SectionCard> from '@/components/ui/SectionCard'.",
  "",
  "4. DO NOT change: Ant Design component props (type, size, shape), dataIndex, key, class names, existing CSS variable usage (var(--*)), logo content.",
  "",
  "Make ALL replacements in one pass."
].join("\n");

phase("Read-&-Fix");
var results = await pipeline(
  files,
  function(f) {
    return agent(FIX_PROMPT, { label: "fix:" + f.split("/").pop(), isolation: "worktree", phase: "Read-&-Fix" });
  }
);

phase("Verify");
var buildResult = await agent(
  "Run: cd frontend-next && npm run build 2>&1 | tail -15. Report PASS or FAIL (count errors).",
  { label: "verify-build", phase: "Verify" }
);

log("Files fixed: " + results.filter(Boolean).length);
log("Build result: " + buildResult);
return { fixed: results.filter(Boolean).length, build: buildResult };
