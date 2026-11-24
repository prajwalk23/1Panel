// @see: http://eslint.cn

module.exports = {
    root: true,
    env: {
        browser: true,
        node: true,
        es6: true,
    },
    /* How to parse the syntax */
    parser: 'vue-eslint-parser',
    /* Lower priority than the parser options */
    parserOptions: {
        parser: '@typescript-eslint/parser',
        ecmaVersion: 2020,
        sourceType: 'module',
        jsxPragma: 'React',
        ecmaFeatures: {
            jsx: true,
        },
    },
    /* Extend existing rule sets */
    extends: [
        'plugin:vue/recommended',
        'plugin:@typescript-eslint/recommended',
        'prettier',
        'plugin:prettier/recommended',
    ],
    /*
     * "off" or 0    ==>  turn the rule off
     * "warn" or 1   ==>  rule reports a warning (does not stop execution)
     * "error" or 2  ==>  rule reports an error
     */
    rules: {
        // eslint (http://eslint.cn/docs/rules)
        'no-var': 'error', // Require let or const instead of var
        'no-multiple-empty-lines': ['error', { max: 1 }], // Disallow multiple empty lines
        'no-use-before-define': 'off', // Disallow use of functions/classes/variables before definition
        'prefer-const': 'off', // Flag let variables never reassigned after initial assignment
        'no-irregular-whitespace': 'off', // Disallow irregular whitespace

        // typeScript (https://typescript-eslint.io/rules)
        '@typescript-eslint/no-unused-vars': ['error', { caughtErrors: 'none' }], // Keep the existing catch(error) style; do not force cleanup on dependency upgrades
        '@typescript-eslint/no-inferrable-types': 'off', // Explicit types that are easily inferred add needless verbosity
        '@typescript-eslint/no-namespace': 'off', // Disallow custom TypeScript modules and namespaces
        '@typescript-eslint/no-explicit-any': 'off', // Disallow the any type
        '@typescript-eslint/ban-ts-ignore': 'off', // Disallow ts-ignore
        '@typescript-eslint/ban-types': 'off', // Disallow specific types
        '@typescript-eslint/explicit-function-return-type': 'off', // Disallow explicit types on variables or parameters initialized to number, string or boolean
        '@typescript-eslint/no-var-requires': 'off', // Disallow require statements in imports
        '@typescript-eslint/no-empty-function': 'off', // Disallow empty functions
        '@typescript-eslint/no-empty-object-type': 'off', // Keep the existing {} type style; no bulk refactor on rule upgrades
        '@typescript-eslint/no-use-before-define': 'off', // Disallow use of variables before definition
        '@typescript-eslint/no-unused-expressions': 'off', // Keep the existing short-circuit expression style
        '@typescript-eslint/no-unsafe-function-type': 'off', // Keep the existing Function type style
        '@typescript-eslint/ban-ts-comment': 'off', // Disallow ts directive comments or require a description after the directive
        '@typescript-eslint/no-non-null-assertion': 'off', // Disallow non-null assertions with the postfix operator (!)
        '@typescript-eslint/explicit-module-boundary-types': 'off', // Require explicit return and argument types on exported functions and public class methods

        // vue (https://eslint.vuejs.org/rules)
        'vue/no-v-html': 'off', // Disallow v-html
        'vue/v-slot-style': 'error', // Enforce v-slot directive style
        'vue/no-mutating-props': 'off', // Disallow mutating component props
        'vue/custom-event-name-casing': 'off', // Enforce casing for custom event names
        'vue/attributes-order': 'off', // Enforce attribute order
        'vue/one-component-per-file': 'off', // Require each component to be in its own file
        'vue/html-closing-bracket-newline': 'off', // Require or disallow a newline before the closing bracket of a tag
        'vue/max-attributes-per-line': 'off', // Enforce max attributes per line
        'vue/multiline-html-element-content-newline': 'off', // Require newlines around the content of multi-line elements
        'vue/singleline-html-element-content-newline': 'off', // Require newlines around the content of single-line elements
        'vue/attribute-hyphenation': 'off', // Enforce attribute naming style on custom components in templates
        'vue/require-default-prop': 'off', // Require default values for non-required props
        'vue/multi-word-component-names': 'off', // Require component names to be multi-word
        'vue/no-dupe-keys': 'off', // Keep the existing same-name script/template usage; no extra rework on dependency upgrades
    },
};
