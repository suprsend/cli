/* import-cleaner.ts */

export type Lang =
    | 'python'
    | 'typescript'
    | 'typescript-zod'
    | 'javascript'
    | 'java'
    | 'kotlin'
    | 'go'
    | 'swift'
    | 'dart';

type ParsedNone = { kind: 'none' };

type ParsedImport = {
    kind: 'import';
    lang: Lang;
    raw: string;
    module?: string;

    // JS/TS
    isTypeOnly?: boolean;
    sideEffect?: boolean;
    defaultName?: string;
    named?: string[];
    namespaceName?: string;

    // Python
    pyFrom?: boolean;
    pyItems?: string[];

    // Go
    goAlias?: string | null;
};

type Parsed = ParsedNone | ParsedImport;

const WS_ONLY = /^[\s;]*$/;

export function parseImportLine(line: string, language: string): Parsed {
    const raw = line.trim();
    const lang = language.toLowerCase() as Lang;

    if (WS_ONLY.test(raw)) return { kind: 'none' };

    switch (lang) {
        case 'typescript':
        case 'typescript-zod':
        case 'javascript': {
            // Side-effect: import 'module';
            const sideEffect = raw.match(/^import\s+['"]([^'"]+)['"]\s*;?$/);
            if (sideEffect) {
                return { kind: 'import', lang, raw, module: sideEffect[1], sideEffect: true };
            }

            // import type { A, B } from 'mod';
            const typeNamed = raw.match(
                /^import\s+type\s+\{([^}]+)\}\s+from\s+['"]([^'"]+)['"]\s*;?$/
            );
            if (typeNamed) {
                const items = typeNamed[1].split(',').map((s) => s.trim()).filter(Boolean);
                return { kind: 'import', lang, raw, module: typeNamed[2], isTypeOnly: true, named: items };
            }

            // import type X from 'mod';
            const typeDefault = raw.match(
                /^import\s+type\s+([A-Za-z_$][A-Za-z0-9_$]*)\s+from\s+['"]([^'"]+)['"]\s*;?$/
            );
            if (typeDefault) {
                return {
                    kind: 'import',
                    lang,
                    raw,
                    module: typeDefault[2],
                    isTypeOnly: true,
                    defaultName: typeDefault[1],
                };
            }

            // import Foo, {A, B as C} from 'mod';
            const defaultAndNamed = raw.match(
                /^import\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*,\s*\{([^}]+)\}\s+from\s+['"]([^'"]+)['"]\s*;?$/
            );
            if (defaultAndNamed) {
                const items = defaultAndNamed[2].split(',').map((s) => s.trim()).filter(Boolean);
                return {
                    kind: 'import',
                    lang,
                    raw,
                    module: defaultAndNamed[3],
                    defaultName: defaultAndNamed[1],
                    named: items,
                };
            }

            // import {A, B as C} from 'mod';
            const named = raw.match(/^import\s+\{([^}]+)\}\s+from\s+['"]([^'"]+)['"]\s*;?$/);
            if (named) {
                const items = named[1].split(',').map((s) => s.trim()).filter(Boolean);
                return { kind: 'import', lang, raw, module: named[2], named: items };
            }

            // import * as NS from 'mod';
            const namespace = raw.match(
                /^import\s+\*\s+as\s+([A-Za-z_$][A-Za-z0-9_$]*)\s+from\s+['"]([^'"]+)['"]\s*;?$/
            );
            if (namespace) {
                return {
                    kind: 'import',
                    lang,
                    raw,
                    module: namespace[2],
                    namespaceName: namespace[1],
                };
            }

            // import Foo from 'mod';
            const defaultOnly = raw.match(
                /^import\s+([A-Za-z_$][A-Za-z0-9_$]*)\s+from\s+['"]([^'"]+)['"]\s*;?$/
            );
            if (defaultOnly) {
                return {
                    kind: 'import',
                    lang,
                    raw,
                    module: defaultOnly[2],
                    defaultName: defaultOnly[1],
                };
            }

            return { kind: 'none' };
        }

        case 'python': {
            // from module import a, b as c
            const fromMatch = raw.match(/^from\s+([A-Za-z0-9_.]+)\s+import\s+(.+)$/);
            if (fromMatch) {
                const module = fromMatch[1];
                const items = fromMatch[2]
                    .split(',')
                    .map((s) => s.trim())
                    .filter(Boolean);
                return {
                    kind: 'import',
                    lang,
                    raw,
                    module,
                    pyFrom: true,
                    pyItems: items,
                };
            }

            // import module [as alias] [, module2 ...]
            const importMatch = raw.match(/^import\s+(.+)$/);
            if (importMatch) {
                const tokens = importMatch[1]
                    .split(',')
                    .map((s) => s.trim())
                    .filter(Boolean);
                return {
                    kind: 'import',
                    lang,
                    raw,
                    module: '__py_import_list__',
                    pyFrom: false,
                    pyItems: tokens,
                };
            }
            return { kind: 'none' };
        }

        case 'go': {
            // import alias "path"
            const alias = raw.match(/^import\s+([A-Za-z0-9_\.]+)\s+"([^"]+)"\s*$/);
            if (alias) {
                return { kind: 'import', lang, raw, module: alias[2], goAlias: alias[1] };
            }
            // import "path"
            const single = raw.match(/^import\s+"([^"]+)"\s*$/);
            if (single) {
                return { kind: 'import', lang, raw, module: single[1], goAlias: null };
            }
            return { kind: 'none' };
        }

        case 'java': {
            const m = raw.match(/^import\s+([^;]+);$/);
            if (m) return { kind: 'import', lang, raw, module: m[1] };
            return { kind: 'none' };
        }

        case 'kotlin': {
            const m = raw.match(/^import\s+(\S+)\s*$/);
            if (m) return { kind: 'import', lang, raw, module: m[1] };
            return { kind: 'none' };
        }

        case 'swift': {
            const m = raw.match(/^import\s+(\S+)\s*$/);
            if (m) return { kind: 'import', lang, raw, module: m[1] };
            return { kind: 'none' };
        }

        case 'dart': {
            const m = raw.match(/^import\s+['"]([^'"]+)['"]\s*;?$/);
            if (m) return { kind: 'import', lang, raw, module: m[1] };
            return { kind: 'none' };
        }
    }

    return { kind: 'none' };
}

export function cleanAndHoistImports(source: string, language: Lang): string {
    const lines = source.split('\n');

    const parsed: Parsed[] = [];
    const originalOrderModules: string[] = [];
    const cleanedLines: string[] = [];
    const leadingComments: string[] = [];
    let firstPackageLine: string | null = null;
    let inLeadingComments = true;

    const pushParsed = (p: Parsed) => {
        if (p.kind === 'none') return false;
        parsed.push(p);
        if (p.module && !originalOrderModules.includes(p.module)) {
            originalOrderModules.push(p.module);
        }
        return true;
    };

    const lang = language.toLowerCase() as Lang;

    let i = 0;
    while (i < lines.length) {
        const line = lines[i];

        if (inLeadingComments && /^\s*(\/\/|#|\/\*)/.test(line)) {
            leadingComments.push(line);
            i++;
            continue;
        }

        if (lang === 'go' && /^\s*package\s+\w+/.test(line)) {
            inLeadingComments = false;
            if (!firstPackageLine) {
                firstPackageLine = line;
            }
            i++;
            continue;
        }

        // Handle Go import blocks
        if (lang === 'go' && /^\s*import\s*\(\s*$/.test(line)) {
            inLeadingComments = false;
            i++;
            while (i < lines.length && !/^\s*\)\s*$/.test(lines[i])) {
                const inner = lines[i].trim();
                if (!WS_ONLY.test(inner)) {
                    // Normalize: prepend 'import ' so our regexes match
                    const normalized = inner.startsWith('import ') ? inner : `import ${inner}`;
                    const p = parseImportLine(normalized, 'go');
                    pushParsed(p);
                }
                i++;
            }
            // skip the closing ')'
            i++;
            continue;
        }

        // Normal single-line import parsing
        const p = parseImportLine(line, language);
        if (!pushParsed(p)) {
            inLeadingComments = false;
            cleanedLines.push(line);
        }
        else {
            inLeadingComments = false;
        }
        i++;
    }

    // Build language-specific deduped import blocks
    let hoisted = '';

    if (lang === 'typescript' || lang === 'typescript-zod' || lang === 'javascript') {
        type TsBucket = {
            sideEffect: boolean;
            defaultName?: string;
            namespaceName?: string;
            named: Set<string>;
            typeNamed: Set<string>;
            typeDefaultName?: string; // rare
            firstSeenIndex: number;
        };

        const buckets = new Map<string, TsBucket>();

        const ensureBucket = (m: string) => {
            if (!buckets.has(m)) {
                buckets.set(m, {
                    sideEffect: false,
                    named: new Set(),
                    typeNamed: new Set(),
                    firstSeenIndex: originalOrderModules.indexOf(m),
                });
            }
            return buckets.get(m)!;
        };

        for (const p of parsed) {
            if (p.kind === 'none' || !p.module) continue;
            const b = ensureBucket(p.module);

            if (p.sideEffect) {
                b.sideEffect = true;
                continue;
            }
            if (p.namespaceName) {
                if (!b.namespaceName) b.namespaceName = p.namespaceName;
                continue;
            }
            if (p.isTypeOnly) {
                if (p.named && p.named.length) p.named.forEach((x) => b.typeNamed.add(x));
                if (p.defaultName && !b.typeDefaultName) b.typeDefaultName = p.defaultName;
                continue;
            }
            if (p.defaultName && !b.defaultName) b.defaultName = p.defaultName;
            if (p.named) p.named.forEach((x) => b.named.add(x));
        }

        // Render buckets in first-seen order
        const ordered = [...buckets.entries()].sort(
            (a, b) => a[1].firstSeenIndex - b[1].firstSeenIndex
        );

        const out: string[] = [];
        for (const [mod, b] of ordered) {
            if (b.sideEffect) out.push(`import '${mod}';`);
            if (b.namespaceName) out.push(`import * as ${b.namespaceName} from '${mod}';`);

            const namedArr = [...b.named];
            const typeNamedArr = [...b.typeNamed];

            if (b.defaultName && namedArr.length) {
                out.push(`import ${b.defaultName}, { ${namedArr.join(', ')} } from '${mod}';`);
            } else if (b.defaultName) {
                out.push(`import ${b.defaultName} from '${mod}';`);
            } else if (namedArr.length) {
                out.push(`import { ${namedArr.join(', ')} } from '${mod}';`);
            }

            if (b.typeDefaultName && typeNamedArr.length) {
                out.push(`import type ${b.typeDefaultName}, { ${typeNamedArr.join(', ')} } from '${mod}';`);
            } else if (b.typeDefaultName) {
                out.push(`import type ${b.typeDefaultName} from '${mod}';`);
            } else if (typeNamedArr.length) {
                out.push(`import type { ${typeNamedArr.join(', ')} } from '${mod}';`);
            }
        }

        if (out.length) hoisted = out.join('\n') + '\n\n';
    } else if (lang === 'python') {
        // Merge plain 'import …' tokens and 'from … import …' items
        const fromMap = new Map<string, Set<string>>(); // module -> items
        const plainTokens = new Set<string>(); // "mod" or "mod as alias"
        const order: string[] = [];

        for (const p of parsed) {
            if (p.kind === 'none') continue;
            if (p.pyFrom && p.module && p.pyItems) {
                if (!fromMap.has(p.module)) {
                    fromMap.set(p.module, new Set());
                    order.push(`from:${p.module}`);
                }
                const set = fromMap.get(p.module)!;
                p.pyItems.forEach((t) => set.add(t));
            } else if (p.pyItems && !p.pyFrom) {
                for (const tok of p.pyItems) {
                    if (!plainTokens.has(tok)) plainTokens.add(tok);
                }
                if (!order.includes('import-list')) order.push('import-list');
            }
        }

        const out: string[] = [];
        // Render plain imports each on its own line for clean diffs
        if (plainTokens.size) {
            for (const tok of [...plainTokens]) out.push(`import ${tok}`);
        }

        // Then render from-imports in first-seen order
        for (const tag of order) {
            if (tag.startsWith('from:')) {
                const mod = tag.slice(5);
                const items = [...(fromMap.get(mod) ?? [])];
                if (items.length) out.push(`from ${mod} import ${items.join(', ')}`);
            }
        }

        if (out.length) hoisted = out.join('\n') + '\n\n';
    } else if (lang === 'go') {
        // Collect modules with first-seen alias
        const seen = new Map<string, string | null>(); // module -> alias|null
        const order: string[] = [];

        for (const p of parsed) {
            if (p.kind === 'none' || !p.module) continue;
            if (!seen.has(p.module)) {
                seen.set(p.module, p.goAlias ?? null);
                order.push(p.module);
            }
        }

        if (order.length === 1) {
            const mod = order[0];
            const alias = seen.get(mod);
            hoisted = alias ? `import ${alias} "${mod}"\n\n` : `import "${mod}"\n\n`;
        } else if (order.length > 1) {
            const body = order
                .map((m) => {
                    const alias = seen.get(m);
                    return alias ? `\t${alias} "${m}"` : `\t"${m}"`;
                })
                .join('\n');
            hoisted = `import (\n${body}\n)\n\n`;
        }
    } else if (lang === 'java') {
        const set = new Set<string>();
        const out: string[] = [];
        for (const p of parsed) {
            if (p.kind === 'none' || !p.module) continue;
            const line = `import ${p.module};`;
            if (!set.has(line)) {
                set.add(line);
                out.push(line);
            }
        }
        if (out.length) hoisted = out.join('\n') + '\n\n';
    } else if (lang === 'kotlin' || lang === 'swift' || lang === 'dart') {
        const set = new Set<string>();
        const out: string[] = [];
        for (const p of parsed) {
            if (p.kind === 'none' || !p.module) continue;
            const line = lang === 'dart' ? `import '${p.module}';` : `import ${p.module}`;
            if (!set.has(line)) {
                set.add(line);
                out.push(line);
            }
        }
        if (out.length) hoisted = out.join('\n') + '\n\n';
    }

    // Remove all original import lines from the body (already skipped in cleanedLines)
    const body = cleanedLines.join('\n').replace(/^\s*\n+/, '');

    // Keep special headers before imports
    const headerLines: string[] = [];
    const bodyLines = body.split('\n');
    while (
        bodyLines.length &&
        (/^#!/.test(bodyLines[0]) || // shebang
            (lang === 'dart' && /^\s*library\s+/.test(bodyLines[0])) ||
            (lang === 'go' && /^\s*package\s+/.test(bodyLines[0])) ||
            (lang === 'swift' && /^\s*@_exported\b/.test(bodyLines[0])))
    ) {
        headerLines.push(bodyLines.shift()!);
    }

    const finalText =
        (leadingComments.length ? leadingComments.join('\n') + '\n\n' : '') +
        (headerLines.length ? headerLines.join('\n') + '\n' : '') +
        ((lang == "go" && firstPackageLine) ? firstPackageLine + '\n\n' : '') +
        hoisted +
        bodyLines.join('\n');

    return finalText.replace(/\n{3,}/g, '\n\n'); // collapse extra blank lines
}

/** Optional convenience wrapper: detect language from filename. */
export function cleanAndHoistImportsForFile(source: string, fileName: string): string {
    const lower = fileName.toLowerCase();
    let lang: Lang;
    if (lower.endsWith('.ts') || lower.endsWith('.tsx')) lang = 'typescript';
    else if (lower.endsWith('.js') || lower.endsWith('.mjs') || lower.endsWith('.cjs')) lang = 'javascript';
    else if (lower.endsWith('.py')) lang = 'python';
    else if (lower.endsWith('.go')) lang = 'go';
    else if (lower.endsWith('.java')) lang = 'java';
    else if (lower.endsWith('.kt') || lower.endsWith('.kts')) lang = 'kotlin';
    else if (lower.endsWith('.swift')) lang = 'swift';
    else if (lower.endsWith('.dart')) lang = 'dart';
    else lang = 'typescript'; // default fallback
    return cleanAndHoistImports(source, lang);
}