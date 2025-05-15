import { TextEncoder } from 'util';

// jsdom, used to run tests, doesn't support text-encoder
// https://github.com/remix-run/react-router/issues/12363

global.TextEncoder = TextEncoder;
