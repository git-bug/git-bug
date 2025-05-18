module.exports = {
  extends: ['react-app'],
  rules: {
    'import/order': [
      'error',
      {
        alphabetize: { order: 'asc' },
        pathGroups: [
          {
            pattern: '@material-ui/**',
            group: 'external',
            position: 'after',
          },
          {
            pattern: '*.generated',
            group: 'sibling',
            position: 'after',
          },
        ],
        pathGroupsExcludedImportTypes: ['builtin'],
        groups: [
          ['builtin', 'external'],
          ['internal', 'parent'],
          ['sibling', 'index'],
        ],
        'newlines-between': 'always',
      },
    ],
  },
  settings: {
    'import/internal-regex': '^src/',
  },
  ignorePatterns: ['**/*.generated.tsx'],

  overrides: [
    {
      files: ['*.graphql'],
      parser: '@graphql-eslint/eslint-plugin',
      plugins: ['@graphql-eslint'],
      rules: {
        '@graphql-eslint/known-type-names': 'error',
      },
      parserOptions: {
        graphQLConfig: {
          schema: './src/schema.json',
          documents: './src/**/*.graphql',
        },
      },
    },
  ],
};
