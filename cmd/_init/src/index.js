// eslint-disable-next-line no-unused-vars
const main = (namespace) => {
  const { router, response, mustache } = namespace;

  // routes
  router.get('/', () => mustache.html('home'));

  if (!router.route(namespace)) {
    response.notFound('Oops');
  }
};