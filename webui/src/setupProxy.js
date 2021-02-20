const { createProxyMiddleware } = require('http-proxy-middleware');
const target = process.env.REACT_APP_PROXY || 'http://localhost:3001';

module.exports = function (app) {
  app.use(
    createProxyMiddleware(['/graphql', '/playground'], { target: target })
  );
};
