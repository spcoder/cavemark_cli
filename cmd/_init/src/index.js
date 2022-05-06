// noinspection JSUnusedGlobalSymbols
const main = (namespace) => {
  const { router, response } = namespace.v1;

  router.get('/', () => response.ok("Let's go!"));

  if (!router.route(namespace)) {
    response.notFound('Not found');
  }
};