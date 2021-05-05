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

  expect(parse(`label:abc freetext`)).toEqual({
    label: [`abc`],
    freetext: [''],
  });

  expect(parse(`label:abc with "quotes" 'in' freetext`)).toEqual({
    label: [`abc`],
    with: [''],
    '"quotes"': [''],
    "'in'": [''],
    freetext: [''],
  });
});

it('parses a quoted query', () => {
  expect(parse(`foo:"bar"`)).toEqual({
    foo: [`"bar"`],
  });

  expect(parse(`foo:'bar'`)).toEqual({
    foo: [`'bar'`],
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

  expect(parse(`label:"with:quoated:colon"`)).toEqual({
    label: [`"with:quoated:colon"`],
  });

  expect(parse(`label:'name ends after this ->' quote'`)).toEqual({
    label: [`'name ends after this ->'`],
    "quote'": [``],
  });

  expect(parse(`label:"name ends after this ->" quote"`)).toEqual({
    label: [`"name ends after this ->"`],
    'quote"': [``],
  });

  expect(parse(`label:'this ->"<- quote belongs to label name'`)).toEqual({
    label: [`'this ->"<- quote belongs to label name'`],
  });

  expect(parse(`label:"this ->'<- quote belongs to label name"`)).toEqual({
    label: [`"this ->'<- quote belongs to label name"`],
  });

  expect(parse(`label:'names end with'whitespace not with quotes`)).toEqual({
    label: [`'names end with'whitespace`],
    not: [``],
    with: [``],
    quotes: [``],
  });

  expect(parse(`label:"names end with"whitespace not with quotes`)).toEqual({
    label: [`"names end with"whitespace`],
    not: [``],
    with: [``],
    quotes: [``],
  });
});

it('should not escape nested quotes', () => {
  expect(parse(`foo:'do not escape this ->'<- quote'`)).toEqual({
    foo: [`'do not escape this ->'<-`],
    "quote'": [``],
  });

  expect(parse(`foo:'do not escape this ->"<- quote'`)).toEqual({
    foo: [`'do not escape this ->"<- quote'`],
  });

  expect(parse(`foo:"do not escape this ->"<- quote"`)).toEqual({
    foo: [`"do not escape this ->"<-`],
    'quote"': [``],
  });

  expect(parse(`foo:"do not escape this ->'<- quote"`)).toEqual({
    foo: [`"do not escape this ->'<- quote"`],
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

it('parses a key:value:value query', () => {
  expect(parse(`meta:github:"https://github.com/MichaelMure/git-bug"`)).toEqual(
    {
      meta: [`github:"https://github.com/MichaelMure/git-bug"`],
    }
  );
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
