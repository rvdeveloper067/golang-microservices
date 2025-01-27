package services

import (
	"github.com/rprajapati0067/golang-microservices/introduction/src/api/config"
	"github.com/rprajapati0067/golang-microservices/introduction/src/api/domain/github"
	"github.com/rprajapati0067/golang-microservices/introduction/src/api/domain/repositories"
	"github.com/rprajapati0067/golang-microservices/introduction/src/api/provider/github_provider"
	"github.com/rprajapati0067/golang-microservices/introduction/src/api/utils/errors"
	"net/http"
	"sync"
)

type reposService struct {
}
type reposServiceInterface interface {
	CreateRepo(request repositories.CreateRepoRequest) (*repositories.CreateRepoResponse, errors.ApiError)
	CreateRepos(request []repositories.CreateRepoRequest) (repositories.CreateReposResponse, errors.ApiError)
}

var (
	RepositoryService reposServiceInterface
)

func init() {
	RepositoryService = &reposService{}
}

func (s *reposService) CreateRepo(input repositories.CreateRepoRequest) (*repositories.CreateRepoResponse, errors.ApiError) {

	if err := input.Validate(); err != nil {
		return nil, err
	}

	request := github.CreateRepoRequest{
		Name:        input.Name,
		Description: input.Description,
		Private:     false,
	}

	response, err := github_provider.CreateRepo(config.GetGithubAccessToken(), request)

	if err != nil {
		return nil, errors.NewApiError(err.StatusCode, err.Message)
	}
	result := repositories.CreateRepoResponse{
		Id:    response.Id,
		Owner: response.Owner.Login,
		Name:  response.Name,
	}
	return &result, nil

}
func (s *reposService) CreateRepos(requests []repositories.CreateRepoRequest) (repositories.CreateReposResponse, errors.ApiError) {
	input := make(chan repositories.CreateRepositoriesResult)
	output := make(chan repositories.CreateReposResponse)

	defer close(output)
	var wg sync.WaitGroup
	go s.handleRepoResults(&wg, input, output)

	// 3 Request to process
	for _, current := range requests {
		wg.Add(1)
		go s.createRepoConcurrent(current, input)
	}
	wg.Wait()
	close(input)
	result := <-output
	successCreation := 0
	for _, current := range result.Results {
		if current.Response != nil {
			successCreation++
		}
	}

	if successCreation == 0 {
		result.StatusCode = result.Results[0].Error.Status()
	} else if successCreation == len(requests) {
		result.StatusCode = http.StatusCreated
	} else {
		result.StatusCode = http.StatusPartialContent
	}

	return result, nil
}

func (s *reposService) handleRepoResults(wg *sync.WaitGroup, input chan repositories.CreateRepositoriesResult, output chan repositories.CreateReposResponse) {
	var results repositories.CreateReposResponse
	for incomingEvent := range input {

		repoResult := repositories.CreateRepositoriesResult{
			Response: incomingEvent.Response,
			Error:    incomingEvent.Error,
		}
		results.Results = append(results.Results, repoResult)
		wg.Done()
	}
	output <- results

}

func (s *reposService) createRepoConcurrent(input repositories.CreateRepoRequest, output chan repositories.CreateRepositoriesResult) {
	if err := input.Validate(); err != nil {
		output <- repositories.CreateRepositoriesResult{Error: err}
		return
	}

	result, err := s.CreateRepo(input)

	if err != nil {
		output <- repositories.CreateRepositoriesResult{Error: err}
		return
	}
	output <- repositories.CreateRepositoriesResult{Response: result}
}
