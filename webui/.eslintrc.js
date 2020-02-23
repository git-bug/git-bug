module.exports = {
  extends: [
    'react-app',
    'prettier/@typescript-eslint',
    'plugin:prettier/recommended',
  ],
  plugins: ['graphql'],
  rules: {
    'graphql/template-strings': [
      'error',
      {
        schemaJson: require('./src/schema.json'),
        env: 'literal',
      },
    ],
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
        pathGroupsExcludedImportTypes: ["builtin"],
        groups: [['builtin', 'external'], ['internal', 'parent'], ['sibling', 'index']],
        'newlines-between': 'always',
      },
    ],
  },
  settings: {
    'import/internal-regex': '^src/',
  },
};
