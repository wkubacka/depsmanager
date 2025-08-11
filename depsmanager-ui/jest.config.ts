import type { Config } from 'jest';

const config: Config = {
  preset: 'jest-preset-angular',
  testEnvironment: 'jsdom',
  setupFilesAfterEnv: ['<rootDir>/src/setup-jest.ts'],
  transform: {
    '^.+\\.(ts|mjs|js|html)$': [
      'jest-preset-angular',
      { tsconfig: '<rootDir>/tsconfig.spec.json', stringifyContentPathRegex: '\\.html$' }
    ]
  },
  moduleFileExtensions: ['ts', 'html', 'js', 'mjs', 'json'],
  testMatch: ['**/?(*.)+(spec).ts']
};

export default config;

