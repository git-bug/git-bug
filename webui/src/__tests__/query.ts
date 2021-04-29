import { parse, stringify, quote } from 'src/pages/list/Filter';

it('parses a simple query', () => {
  expect(parse('foo:bar')).toEqual({
    foo: ['bar'],
  });
});

it('parses a query with multiple filters', () => {
  expect(parse(`foo:bar baz:foo-bar`)).toEqual({
    foo: ['bar'],
    baz: ['foo-bar'],
  });
});

it('parses a quoted query', () => {
  expect(parse(`foo:"bar"`)).toEqual({
    foo: [`"bar"`],
  });

  expect(parse(`foo:'bar'`)).toEqual({
    foo: [`'bar'`],
  });

  expect(parse(`label:abc freetext`)).toEqual({
    label: [`abc`],
    freetext: [''],
  });

  expect(parse(`label:abc with "quotes" in freetext`)).toEqual({
    label: [`abc`],
    with: [''],
    quotes: [''],
    in: [''],
    freetext: [''],
  });

  expect(parse(`label:'multi word label'`)).toEqual({
    label: [`'multi word label'`],
  });

  expect(parse(`label:"multi word label"`)).toEqual({
    label: [`"multi word label"`],
  });

  expect(parse(`label:'multi word label with "nested" quotes'`)).toEqual({
    label: [`'multi word label with "nested" quotes'`],
  });

  expect(parse(`label:"multi word label with 'nested' quotes"`)).toEqual({
    label: [`"multi word label with 'nested' quotes"`],
  });

  expect(parse(`foo:'escaped\\' quotes'`)).toEqual({
    foo: [`'escaped\\' quotes'`],
  });
});

it('parses a query with repetitions', () => {
  expect(parse(`foo:bar foo:baz`)).toEqual({
    foo: ['bar', 'baz'],
  });
});

it('parses a complex query', () => {
  expect(parse(`foo:bar foo:baz baz:"foobar" idont:'know'`)).toEqual({
    foo: ['bar', 'baz'],
    baz: [`"foobar"`],
    idont: [`'know'`],
  });
});

it('quotes values', () => {
  expect(quote(`foo`)).toEqual(`foo`);
  expect(quote(`foo bar`)).toEqual(`"foo bar"`);
  expect(quote(`foo "bar"`)).toEqual(`"foo "bar""`);
  expect(quote(`foo 'bar'`)).toEqual(`"foo "bar""`);
  expect(quote(`'foo'`)).toEqual(`"foo"`);
  expect(quote(`foo "bar" 'baz'`)).toEqual(`"foo "bar" "baz""`);
});

it('stringifies params', () => {
  expect(stringify({ foo: ['bar'] })).toEqual('foo:bar');
  expect(stringify({ foo: ['bar baz'] })).toEqual('foo:"bar baz"');
  expect(stringify({ foo: ['bar', 'baz'] })).toEqual('foo:bar foo:baz');
  expect(stringify({ foo: ['bar'], baz: ['foobar'] })).toEqual(
    'foo:bar baz:foobar'
  );
});
