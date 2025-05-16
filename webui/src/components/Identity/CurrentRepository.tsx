import { useCurrentIdentityQuery } from './CurrentIdentity.generated';

// same as in multi_repo_cache.go
const defaultRepoName = '__default';

const CurrentRepository = (props: { default: string }) => {
  const { loading, error, data } = useCurrentIdentityQuery();

  if (error || loading || !data?.repository?.name) return null;

  let name = data.repository.name;
  if (name === defaultRepoName) {
    name = props.default;
  }

  return <>{name}</>;
};

export default CurrentRepository;
