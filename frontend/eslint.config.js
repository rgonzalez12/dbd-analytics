import js from '@eslint/js';
import svelte from 'eslint-plugin-svelte';
import prettier from 'eslint-config-prettier';
import ts from '@typescript-eslint/eslint-plugin';
import parser from '@typescript-eslint/parser';
import svelteParser from 'svelte-eslint-parser';

export default [
	js.configs.recommended,
	...svelte.configs['flat/recommended'],
	prettier,
	{
		languageOptions: {
			parser: parser,
			parserOptions: {
				ecmaVersion: 2022,
				sourceType: 'module'
			}
		},
		plugins: {
			'@typescript-eslint': ts
		},
		rules: {
			'@typescript-eslint/no-unused-vars': 'error'
		}
	},
	{
		files: ['**/*.svelte'],
		languageOptions: {
			parser: svelteParser,
			parserOptions: {
				parser: parser,
				extraFileExtensions: ['.svelte']
			}
		}
	},
	{
		ignores: ['build/', '.svelte-kit/', 'dist/']
	}
];
