// eslint-disable-next-line no-unused-vars
interface namespace {
    v1: v1
}

interface v1 {
    response: Response;
    request: Request;
    router: Router;
    db(connStr: string): DatabaseConnection;
    validate: Validate;
    crypto: Cryptography;
    mail: Mail;
    money: Money;
    mustache: Mustache;
}

interface Router {
    useStatic(): Router;

    get(path: string, handler: (namespace: never) => void): Router;

    get(path: string, middleware: (namespace: never) => void, handler: (namespace: never) => void): Router;

    post(path: string, handler: (namespace: never) => void): Router;

    post(path: string, middleware: (namespace: never) => void, handler: (namespace: never) => void): Router;

    put(path: string, handler: (namespace: never) => void): Router;

    put(path: string, middleware: (namespace: never) => void, handler: (namespace: never) => void): Router;

    patch(path: string, handler: (namespace: never) => void): Router;

    patch(path: string, middleware: (namespace: never) => void, handler: (namespace: never) => void): Router;

    delete(path: string, handler: (namespace: never) => void): Router;

    delete(path: string, middleware: (namespace: never) => void, handler: (namespace: never) => void): Router;

    options(path: string, handler: (namespace: never) => void): Router;

    options(path: string, middleware: (namespace: never) => void, handler: (namespace: never) => void): Router;

    route(namespace: never): boolean;
}

interface Mail {
    connect(host: string, auth: boolean, port: number, username: string, password: string): MailSession;
}

interface Money {
    formatted(value: number): string;

    formatted(value: number, language: string): string;

    formatted(value: number, language: string, country: string): string;
}

interface MailSession {
    send(from: string, to: string, subject: string, body: string): void;
}

interface Cryptography {
    hashPassword(value: string): HashPasswordResult;

    hashPassword(value: string, iterationCount: number, keyLength: number): HashPasswordResult;

    randomInteger(min: number, max: number): number;

    confirmationCode(): number;
}

interface HashPasswordResult {
    salt: string;
    passwordHash: string;
    iterationCount: number;
    keyLength: number;
}

interface Validate {
    that(key: string, value: string): Validate;

    isRequired(message?: string): Validate;

    isBetween(min: number, max: number, message?: string): Validate;

    isEmail(message?: String): Validate;

    hasLower(message?: String): Validate;

    hasUpper(message?: String): Validate;

    hasDigit(message?: String): Validate;

    hasSpecial(message?: String): Validate;

    hasSpecial(specialCharacters: string, message?: String): Validate;

    msg(message: string): Validate;

    check(): Object;
}

interface RequestFormData {
    get(name: string): string;

    getAll(name: string): string[];
}

interface Request {
    // @ts-ignore
    method: string;
    path: string;
    fullURL: string;
    body: string;
    form: RequestFormData;

    queryValue(string, never): never

    queryValues(string, never): never[]
}

interface Response {
    // @ts-ignore
    status(value: number): Response;

    body(value: string): Response;

    addHeader(key: string, value: string): Response;

    notFound(body: string | Object): void;

    // @ts-ignore
    ok(body: string | Object): void;

    noContent(): void;

    internalServerError(body: string | Object): void;

    badRequest(body: string | Object): void;

    created(location: string | Object): void;

    redirect(location: string): void;

    removeCookie(name: string): Response;
}

interface DatabaseConnection {
    statement(sql: string): DatabaseStatement;
    statement(): DatabaseStatement;
}

interface DatabaseStatement {
    setSQL(s: string): DatabaseStatement;

    setString(s: string): DatabaseStatement;

    setNumber(n: number): DatabaseStatement;

    setDate(d: Date): DatabaseStatement;

    setBoolean(b: Boolean): DatabaseStatement;

    execute(): DatabaseResult;

    query(): Array<any>;

    queryOne(): Object;

    queryScalar(): any;
}

interface DatabaseResult {
    rowsAffected: number;
    lastInsertId: string;
}

interface Mustache {
    /**
     * returns a string after compiling and running a template.
     * @param name the name of the template found in resources
     * @param scope the object used in the template
     * @returns a string of the result
     * @description
     * note: the name does not need an extension when locating a resource.
     * @example
     * namespace.mustache.execute('account', { title: 'Hello' });
     */
    execute(name: string, scope?: any): string;

    /**
     * compiles and runs a template that is then used for a http response.
     * @param name the name of the template found in resources
     * @param scope the object used in the template
     * @description
     * the name does not need an extension when locating a resource.
     * @example
     * namespace.mustache.execute('account', { title: 'Hello' });
     */
    html(name: string, scope?: any): void;
}
